package browser

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"

	logging "github.com/KonishchevDmitry/go-easy-logging"
	"github.com/chromedp/chromedp"
)

func Configure(ctx context.Context) (_ context.Context, _ func(), retErr error) {
	logging.L(ctx).Debugf("Configuring the browser...")

	browserCtx, stop := configure(ctx)
	defer func() {
		if retErr != nil {
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
		browserCtx, stop = configure(ctx, chromedp.UserAgent(requiredUserAgent))

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
	Status      int
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
		Status:      int(response.Status),
		StatusText:  response.StatusText,
		ContentType: contentType,
		Text:        text,
		HTML:        html,
	}, nil
}

func configure(ctx context.Context, options ...chromedp.ExecAllocatorOption) (_ context.Context, _ func()) {
	ctx, cancelAllocator := chromedp.NewExecAllocator(ctx, slices.Concat(
		chromedp.DefaultExecAllocatorOptions[:],
		options,
	)...)

	ctx, cancelContext := chromedp.NewContext(ctx, chromedp.WithLogf(func(format string, args ...any) {
		logging.L(ctx).Debugf("Browser: "+format, args...)
	}))

	return ctx, func() {
		logging.L(ctx).Debugf("Stopping the browser...")

		if err := chromedp.Cancel(ctx); err != nil {
			logging.L(ctx).Errorf("Failed to stop the browser: %s.", err)
		} else {
			logging.L(ctx).Debugf("The browser has stopped.")
		}

		cancelContext()
		cancelAllocator()
	}
}

var userAgentRe = regexp.MustCompile(
	`^(Mozilla/[^ ]+ \(Macintosh; [^)]+\) AppleWebKit/[^ ]+ \(KHTML, like Gecko\) )([^ ]+)(/[^ ]+ Safari/[^ ]+)$`)

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
	} else if response.Status != http.StatusOK {
		return "", fmt.Errorf(
			"failed to check browser User-Agent: the server returned %d %s",
			response.Status, response.StatusText)
	}

	return response.Text, nil
}
