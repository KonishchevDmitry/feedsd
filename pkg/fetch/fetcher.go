package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"time"

	logging "github.com/KonishchevDmitry/go-easy-logging"
)

func fetch[T any](ctx context.Context, url string, allowedMediaTypes []string, parser func(body io.Reader) (T, error)) (_ T, retErr error) {
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("failed to fetch %s: %w", url, retErr)
		}
	}()

	var zero T

	client := http.Client{
		Timeout: time.Minute,
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return zero, err
	}
	request.Header.Add("User-Agent", "github.com/KonishchevDmitry/newslib")

	response, err := client.Do(request)
	if err != nil {
		return zero, makeTemporaryError(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			logging.L(ctx).Errorf("Failed to close HTTP client body: %s.", err)
		}
	}()

	if statusCode := response.StatusCode; statusCode != http.StatusOK {
		err := fmt.Errorf("the server returned an error: %d %s", statusCode, response.Status)
		if statusCode >= 500 && statusCode < 600 {
			err = makeTemporaryError(err)
		}
		return zero, err
	}

	if err := checkContentType(response, allowedMediaTypes); err != nil {
		return zero, err
	}

	return parser(bodyReader{body: response.Body})
}

type bodyReader struct {
	body io.Reader
}

func (r bodyReader) Read(buf []byte) (int, error) {
	n, err := r.body.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		err = makeTemporaryError(err)
	}
	return n, err
}

func checkContentType(response *http.Response, allowedMediaTypes []string) error {
	contentType := response.Header.Get("Content-Type")

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
