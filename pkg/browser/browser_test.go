package browser

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/KonishchevDmitry/feedsd/pkg/rss"
	"github.com/KonishchevDmitry/feedsd/pkg/test"
	"github.com/KonishchevDmitry/feedsd/pkg/url"
	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	t.Parallel()

	ctx, stop, err := Configure(test.Context(t))
	require.NoError(t, err)
	defer stop()

	var waitGroup sync.WaitGroup
	defer waitGroup.Wait()

	run := func(t *testing.T, name string, test func(t *testing.T)) {
		// XXX(konishchev): Support
		// waitGroup.Go(func() {
		t.Run(name, test)
		// })
	}

	testCases := []struct {
		name        string
		status      int
		contentType string
		body        string
		text        string
		html        string
	}{{
		name:        "text",
		status:      http.StatusOK,
		contentType: "text/plain",
		body:        "Some & text",
		text:        "Some & text",
	}, {
		name:        "error",
		status:      http.StatusInternalServerError,
		contentType: "text/html",
		body:        "Some error",
		text:        "Some error",
		html:        "<html><head></head><body>Some error</body></html>",
	}, {
		name:        "html",
		status:      http.StatusOK,
		contentType: "text/html",
		body:        "<html><body>Some text</body></html>",
		text:        "Some text",
		html:        "<html><head></head><body>Some text</body></html>",
	}, {
		name:        "rss",
		status:      http.StatusOK,
		contentType: rss.ContentType,
		body: heredoc.Doc(`
			<?xml version="1.0" encoding="UTF-8"?>
			<rss version="2.0">
				<channel>
					<title>Feed title</title>
					<link>http://example.com/</link>
					<description>Feed description</description>
				</channel>
			</rss>
		`),
		text: heredoc.Doc(`
			<?xml version="1.0" encoding="UTF-8"?>
			<rss version="2.0">
				<channel>
					<title>Feed title</title>
					<link>http://example.com/</link>
					<description>Feed description</description>
				</channel>
			</rss>
		`),
	}}

	for _, testCase := range testCases {
		run(t, testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", testCase.contentType)
				w.WriteHeader(testCase.status)
				_, _ = io.WriteString(w, testCase.body)
			}))
			defer server.Close()

			response, err := Get(ctx, url.MustParse(server.URL))
			require.NoError(t, err)

			require.Equal(t, server.URL+"/", response.URL)
			require.Equal(t, testCase.status, response.Status)
			require.Equal(t, testCase.contentType, response.ContentType)
			require.Equal(t, testCase.text, response.Text)

			if len(testCase.html) != 0 {
				require.Equal(t, testCase.html, response.HTML)
			}
		})
	}

	run(t, "connection refused", func(t *testing.T) {
		socket, err := net.Listen("tcp6", "[::1]:0")
		require.NoError(t, err)

		url := url.URL{
			Scheme: "http",
			Host:   socket.Addr().String(),
			Path:   "/",
		}
		require.NoError(t, socket.Close())

		_, err = Get(ctx, &url)
		require.ErrorContains(t, err, "page load error net::ERR_CONNECTION_REFUSED")
	})

	run(t, "connection timeout", func(t *testing.T) {
		socket, err := net.Listen("tcp6", "[::1]:0")
		require.NoError(t, err)
		defer func() {
			require.NoError(t, socket.Close())
		}()

		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		_, err = Get(ctx, &url.URL{
			Scheme: "http",
			Host:   socket.Addr().String(),
			Path:   "/",
		})
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})
}

func TestUserAgentRegex(t *testing.T) {
	t.Parallel()

	for _, userAgent := range []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36",
	} {
		t.Run(userAgent, func(t *testing.T) {
			matches := userAgentRe.FindStringSubmatch(userAgent)
			require.NotEmpty(t, matches)
			require.Equal(t, "Chrome", matches[2])
		})
	}
}
