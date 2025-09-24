package browser

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"unicode"

	"al.essio.dev/pkg/shellescape"
	"github.com/KonishchevDmitry/feedsd/internal/util"
	"github.com/KonishchevDmitry/feedsd/pkg/rss"
	logging "github.com/KonishchevDmitry/go-easy-logging"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/samber/mo"
)

const (
	screenWidth, screenHeight     = 1728, 1117
	viewportWidth, viewportHeight = 1664, 992
)

func Configure(ctx context.Context, opts ...Option) (_ context.Context, _ func(), retErr error) {
	logging.L(ctx).Debugf("Configuring the browser...")

	options, err := getOptions(opts)
	if err != nil {
		return ctx, nil, err
	}

	browserCtx, stop, err := configure(ctx, options)
	if err != nil {
		return ctx, nil, err
	}
	defer func() {
		if retErr != nil && stop != nil {
			stop()
		}
	}()

	actualUserAgent, err := getUserAgent(browserCtx)
	if err != nil {
		return ctx, nil, err
	}

	requiredBrowserName := "Chrome"
	if match := userAgentRe.FindStringSubmatch(actualUserAgent); len(match) == 0 ||
		(match[2] != requiredBrowserName && match[2] != "Headless"+requiredBrowserName) {
		return ctx, nil, fmt.Errorf("the browser has an unexpected User-Agent: %q", actualUserAgent)
	} else if match[2] != requiredBrowserName {
		requiredUserAgent := fmt.Sprintf("%s%s%s", match[1], requiredBrowserName, match[3])
		if options.remote.IsPresent() {
			return ctx, nil, fmt.Errorf(
				"the browser has an invalid User-Agent: %q vs %q expected",
				actualUserAgent, requiredUserAgent)
		}

		logging.L(ctx).Debugf("Changing browser User-Agent to %q...", requiredUserAgent)

		stop()
		browserCtx, stop, err = configure(ctx, options, chromedp.UserAgent(requiredUserAgent))
		if err != nil {
			return ctx, nil, err
		}

		actualUserAgent, err = getUserAgent(browserCtx)
		if err != nil {
			return ctx, nil, err
		} else if actualUserAgent != requiredUserAgent {
			return ctx, nil, fmt.Errorf(
				"failed to change browser User-Agent to %q: it still has %q",
				requiredUserAgent, actualUserAgent)
		}
	}

	return browserCtx, stop, nil
}

type Response struct {
	URL         string
	StatusCode  int
	StatusText  string
	ContentType string
	Body        string
}

func Get(ctx context.Context, url *url.URL, opts ...QueryOption) (*Response, error) {
	if context := chromedp.FromContext(ctx); context == nil || context.Browser == nil {
		return nil, errors.New("the browser is not configured")
	}

	var options queryOptions
	for _, opt := range opts {
		opt(&options)
	}

	// We need to create a child context to be able to use browser concurrently
	ctx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	actions := []chromedp.Action{
		&emulation.SetDeviceMetricsOverrideParams{
			Width:  viewportWidth,
			Height: viewportHeight,

			ScreenWidth:  screenWidth,
			ScreenHeight: screenHeight,
		},
		chromedp.Navigate(url.String()),
		chromedp.WaitVisible("body", chromedp.ByQuery),
	}

	if duration := options.sleep; duration != 0 {
		actions = append(actions, chromedp.Sleep(duration))
	}

	var body, html string
	actions = append(actions,
		// FIXME(konishchev): Try https://github.com/chromedp/examples/blob/master/download_image/main.go
		chromedp.Evaluate("document.body.innerText", &body),
		chromedp.OuterHTML("html", &html),
	)

	response, err := chromedp.RunResponse(ctx, actions...)
	if err != nil {
		return nil, err
	}

	var contentType string
	for name, value := range response.Headers {
		if strings.ToLower(name) == "content-type" {
			if value, ok := value.(string); ok {
				contentType = value
				break
			}
		}
	}
	if contentType == "" {
		return nil, errors.New("the server returned a response without Content-Type")
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("the server returned an invalid Content-Type: %q", contentType)
	}

	if mediaType == "text/html" {
		body = html
	} else if slices.Contains(rss.PossibleContentTypes, contentType) {
		prefix := "This XML file does not appear to have any style information associated with it. The document tree is shown below."
		if trimmed := strings.TrimLeftFunc(body, unicode.IsSpace); strings.HasPrefix(trimmed, prefix) {
			body = strings.TrimLeftFunc(trimmed[len(prefix):], unicode.IsSpace)
		}
	}

	return &Response{
		URL:         response.URL,
		StatusCode:  int(response.Status),
		StatusText:  response.StatusText,
		ContentType: contentType,
		Body:        body,
	}, nil
}

func configure(
	ctx context.Context, options options, execOptions ...chromedp.ExecAllocatorOption,
) (_ context.Context, _ func(), retErr error) {
	if chromedp.FromContext(ctx) != nil {
		return ctx, nil, errors.New("an attempt to configure browser when it's already configured")
	}

	var closers []func()
	stop := func() {
		if len(closers) == 0 {
			return
		}

		logging.L(ctx).Debugf("Stopping the browser...")

		// FIXME(konishchev): Cancel on success?
		// It would be better to gracefully stop the browser first via chromedp.Cancel(), but it's buggy and
		// cancelContext() panics after chromedp.Cancel() in case when browser has failed to start.

		for _, close := range slices.Backward(closers) {
			close()
		}

		logging.L(ctx).Debugf("The browser has stopped.")
	}
	defer func() {
		if retErr != nil {
			stop()
		}
	}()

	var (
		allocatorContext context.Context
		cancelAllocator  func()
	)

	if remote, ok := options.remote.Get(); ok {
		if len(execOptions) != 0 {
			return ctx, nil, errors.New("an attempt to pass exec options to remote browser")
		}

		allocatorContext, cancelAllocator = chromedp.NewRemoteAllocator(ctx, remote)
		closers = append(closers, cancelAllocator)
	} else {
		userDataDir, removeUserDataDir, err := getUserDataDir(options.persistentData)
		if err != nil {
			return ctx, nil, err
		}
		closers = append(closers, func() {
			removeUserDataDir(ctx)
		})

		// See https://peter.sh/experiments/chromium-command-line-switches/ for option docs.
		//
		// Bot detection tools:
		// * https://fingerprint-scan.com/
		// * https://bot.sannysoft.com/
		//
		// Don't use flag wrappers, because they may implicitly enable other flags (like chromedp.Headless does).
		allocatorOptions := []chromedp.ExecAllocatorOption{
			// A subset of chromedp.DefaultExecAllocatorOptions which may be actual for us

			chromedp.Flag("no-first-run", true),
			chromedp.Flag("disable-breakpad", true),
			chromedp.Flag("metrics-recording-only", true),
			chromedp.Flag("no-default-browser-check", true),

			chromedp.Flag("mute-audio", true),
			chromedp.Flag("disable-background-networking", true),

			chromedp.Flag("disable-extensions", true),
			chromedp.Flag("disable-default-apps", true),

			chromedp.Flag("safebrowsing-disable-auto-update", true),
			chromedp.Flag("disable-client-side-phishing-detection", true),

			chromedp.Flag("disable-hang-monitor", true),
			chromedp.Flag("disable-renderer-backgrounding", true),
			chromedp.Flag("disable-ipc-flooding-protection", true),
			chromedp.Flag("disable-background-timer-throttling", true),
			chromedp.Flag("disable-backgrounding-occluded-windows", true),

			chromedp.Flag("disable-features", "site-per-process,Translate"),
			chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),

			chromedp.Flag("use-mock-keychain", true),

			// Our customizations

			chromedp.Flag("user-data-dir", userDataDir),

			chromedp.Flag("headless", !options.headful),
			chromedp.Flag("start-maximized", options.headful),

			// https://developer.mozilla.org/en-US/docs/Web/API/Navigator/webdriver
			chromedp.Flag("disable-blink-features", "AutomationControlled"),

			chromedp.ModifyCmdFunc(func(cmd *exec.Cmd) {
				logging.L(ctx).Debugf("Starting the browser: %s", shellescape.QuoteCommand(
					append([]string{cmd.Path}, cmd.Args...)))
			}),
		}

		if util.IsContainer() {
			allocatorOptions = append(allocatorOptions,
				// FIXME(konishchev): disable-dev-shm-usage?
				chromedp.Flag("no-sandbox", true),
				chromedp.Flag("use-gl", "angle"),
				chromedp.Flag("use-angle", "swiftshader"))
		}

		allocatorOptions = append(allocatorOptions, execOptions...)

		allocatorContext, cancelAllocator = chromedp.NewExecAllocator(ctx, allocatorOptions...)
		closers = append(closers, cancelAllocator)
	}

	browserCtx, cancelBrowser := chromedp.NewContext(allocatorContext, chromedp.WithLogf(func(format string, args ...any) {
		logging.L(ctx).Debugf("Browser: "+format, args...)
	}))
	closers = append(closers, cancelBrowser)

	// [ Start the browser ], connect to it and initialize the context
	if err := chromedp.Run(browserCtx); err != nil {
		return ctx, nil, err
	}

	return browserCtx, stop, nil
}

func getUserDataDir(persistent mo.Option[string]) (string, func(ctx context.Context), error) {
	if daemonName, ok := persistent.Get(); ok {
		path, err := getPersistentUserDataDir(daemonName)
		if err != nil {
			return "", nil, err
		}
		return path, func(ctx context.Context) {}, nil
	}

	dataDir, err := os.MkdirTemp("/var/tmp", "feedsd-browser-*")
	if err != nil {
		return "", nil, err
	}

	return dataDir, func(ctx context.Context) {
		// chromedp fails to wait for all browser processes termination, so they can still exist and write to the directory
		// (see https://github.com/chromedp/chromedp/blob/422fa06290cda228e5712bdda55fbf7a0f6c8466/allocate.go#L227),
		// so try to workaround the issue.

		var attempt int

		for {
			attempt++

			if err := os.RemoveAll(dataDir); err != nil {
				message := fmt.Sprintf("Failed to delete browser data directory %q: %s.", dataDir, err)

				var errno syscall.Errno
				if errors.As(err, &errno) && errno == syscall.ENOTEMPTY && attempt < 10 {
					logging.L(ctx).Debugf("%s Retrying...", message)
					continue
				}

				logging.L(ctx).Error(message)
			}

			break
		}

	}, nil
}

// Please note that Chrome has tricky rules for deriving cache directory path from user data path.
//
// See https://chromium.googlesource.com/chromium/src/+/main/docs/user_data_dir.md for details.
func getPersistentUserDataDir(daemonName string) (string, error) {
	basePath := "/var/lib"

	if runtime.GOOS == "darwin" {
		homePath, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		basePath = filepath.Join(homePath, ".cache")
	}

	return filepath.Join(basePath, fmt.Sprintf("%s-browser", daemonName)), nil
}

var userAgentRe = regexp.MustCompile(
	`^(Mozilla/[^ ]+ \([^)]+\) AppleWebKit/[^ ]+ \(KHTML, like Gecko\) )([^ ]+)(/[^ ]+ Safari/[^ ]+)$`)

func getUserAgent(ctx context.Context) (_ string, retErr error) {
	var serverChan chan error
	defer func() {
		if serverChan != nil {
			if err := <-serverChan; err != nil && retErr == nil {
				retErr = fmt.Errorf("test HTTP server has crashed: %w", err)
			}
		}
	}()

	server := http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(r.Header.Get("User-Agent")))
		}),
	}
	defer func() {
		if err := server.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("failed to close test HTTP server: %w", err)
		}
	}()

	socket, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		return "", err
	}
	url := url.URL{
		Scheme: "http",
		Host:   socket.Addr().String(),
		Path:   "/",
	}
	serverChan = make(chan error)
	go func() {
		err := server.Serve(socket)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverChan <- err
	}()

	response, err := Get(ctx, &url)
	if err != nil {
		return "", err
	} else if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"failed to check browser User-Agent: the server returned %d %s",
			response.StatusCode, response.StatusText)
	}

	return response.Body, nil
}
