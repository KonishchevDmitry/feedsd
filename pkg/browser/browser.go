package browser

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/KonishchevDmitry/feedsd/internal/util"
	logging "github.com/KonishchevDmitry/go-easy-logging"
	"github.com/chromedp/chromedp"
)

func Configure(ctx context.Context) (_ context.Context, _ func(), retErr error) {
	logging.L(ctx).Debugf("Configuring the browser...")

	browserCtx, stop, err := configure(ctx)
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
		logging.L(ctx).Debugf("Changing browser User-Agent to %q...", requiredUserAgent)

		stop()
		browserCtx, stop, err = configure(ctx, chromedp.UserAgent(requiredUserAgent))
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

	if err := chromedp.Run(browserCtx); err != nil {
		return ctx, nil, err
	}

	return browserCtx, stop, nil
}

type Response struct {
	URL         string
	StatusCode  int
	StatusText  string
	ContentType string
	Text        string
	HTML        string
}

func Get(ctx context.Context, url *url.URL) (*Response, error) {
	var text, html string
	response, err := chromedp.RunResponse(ctx,
		chromedp.Navigate(url.String()),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Evaluate("document.body.innerText", &text),
		chromedp.OuterHTML("html", &html),
	)
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

	return &Response{
		URL:         response.URL,
		StatusCode:  int(response.Status),
		StatusText:  response.StatusText,
		ContentType: contentType,
		Text:        text,
		HTML:        html,
	}, nil
}

func configure(ctx context.Context, options ...chromedp.ExecAllocatorOption) (_ context.Context, _ func(), retErr error) {
	var closers []func()
	stop := func() {
		logging.L(ctx).Debugf("Stopping the browser...")

		// It would be better to gracefully stop the browser first via chromedp.Cancel(), but it's buggy and
		// cancelContext() panics after chromedp.Cancel() in case when browser has failed to start.

		for _, close := range slices.Backward(closers) {
			close()
		}
	}
	defer func() {
		if retErr != nil {
			stop()
		}
	}()

	dataDir, err := os.MkdirTemp("/var/tmp", "feedsd-browser-*")
	if err != nil {
		return ctx, nil, err
	}
	closers = append(closers, func() {
		// FIXME(konishchev): Failed to delete browser data directory "/var/tmp/feedsd-browser-3627646949": unlinkat /var/tmp/feedsd-browser-3627646949/Default: directory not empty
		if err := os.RemoveAll(dataDir); err != nil {
			logging.L(ctx).Errorf("Failed to delete browser data directory %q: %s", dataDir, err)
		}
	})

	allocatorOptions := slices.Clone(chromedp.DefaultExecAllocatorOptions[:])
	allocatorOptions = append(allocatorOptions, chromedp.UserDataDir(dataDir))
	if util.IsContainer() {
		allocatorOptions = append(allocatorOptions,
			chromedp.NoSandbox,
			chromedp.Flag("use-gl", "angle"),
			chromedp.Flag("use-angle", "swiftshader"))
	}
	allocatorOptions = append(allocatorOptions, options...)

	ctx, cancelAllocator := chromedp.NewExecAllocator(ctx, allocatorOptions...)
	closers = append(closers, cancelAllocator)

	ctx, cancelContext := chromedp.NewContext(ctx, chromedp.WithLogf(func(format string, args ...any) {
		logging.L(ctx).Debugf("Browser: "+format, args...)
	}))
	closers = append(closers, cancelContext)

	return ctx, stop, nil
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

	return response.Text, nil
}
