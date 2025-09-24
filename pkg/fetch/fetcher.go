package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/KonishchevDmitry/feedsd/pkg/browser"
	logging "github.com/KonishchevDmitry/go-easy-logging"
)

func fetch[T any](
	ctx context.Context, url *url.URL, allowedMediaTypes []string, parser func(body io.Reader) (T, error),
	opts ...Option,
) (_ T, retErr error) {
	var zero T
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("failed to fetch %s: %w", url, retErr)
		}
	}()

	var options options
	for _, opt := range opts {
		opt(&options)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	fetchCtx, err := getContext(ctx)
	if err != nil {
		return zero, err
	}

	logging.L(ctx).Debugf("Fetching %s (emulate browser = %v)...", url, options.emulateBrowser.IsPresent())

	var response *fetchResult
	startTime := time.Now()
	if queryOptions, ok := options.emulateBrowser.Get(); ok {
		response, err = browserFetch(ctx, url, queryOptions...)
	} else {
		response, err = httpClientFetch(ctx, url)
	}
	fetchCtx.duration.Observe(time.Since(startTime).Seconds())
	if err != nil {
		return zero, err
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			logging.L(ctx).Errorf("Failed to close HTTP client body: %s.", err)
		}
	}()

	if statusCode := response.StatusCode; statusCode != http.StatusOK {
		err := fmt.Errorf("the server returned an error: %d %s", statusCode, response.StatusText)
		if statusCode >= 500 && statusCode < 600 {
			err = makeTemporaryError(err)
		}
		return zero, err
	}

	if err := checkContentType(response.ContentType, allowedMediaTypes); err != nil {
		return zero, err
	}

	return parser(bodyReader{body: response.Body})
}

type fetchResult struct {
	StatusCode  int
	StatusText  string
	ContentType string
	Body        io.ReadCloser
}

func httpClientFetch(ctx context.Context, url *url.URL) (*fetchResult, error) {
	client := http.Client{}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add("User-Agent", "github.com/KonishchevDmitry/feedsd")

	response, err := client.Do(request)
	if err != nil {
		return nil, makeTemporaryError(err)
	}

	return &fetchResult{
		StatusCode:  response.StatusCode,
		StatusText:  response.Status,
		ContentType: response.Header.Get("Content-Type"),
		Body:        response.Body,
	}, nil
}

func browserFetch(ctx context.Context, url *url.URL, options ...browser.QueryOption) (*fetchResult, error) {
	response, err := browser.Get(ctx, url, options...)
	if err != nil {
		return nil, makeTemporaryError(err)
	}

	return &fetchResult{
		StatusCode:  response.StatusCode,
		StatusText:  response.StatusText,
		ContentType: response.ContentType,
		Body:        io.NopCloser(strings.NewReader(response.Body)),
	}, nil
}

type bodyReader struct {
	body io.Reader
}

var _ io.Reader = bodyReader{}

func (r bodyReader) Read(buf []byte) (int, error) {
	n, err := r.body.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		err = makeTemporaryError(err)
	}
	return n, err
}

func checkContentType(contentType string, allowedMediaTypes []string) error {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf("got an invalid Content-Type: %w", err)
	}

	for _, allowedMediaType := range allowedMediaTypes {
		if mediaType == allowedMediaType {
			return nil
		}
	}

	return fmt.Errorf("got an invalid Content-Type (%s)", mediaType)
}
