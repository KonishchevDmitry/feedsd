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
	"github.com/KonishchevDmitry/feedsd/pkg/test/testutil"
	"github.com/KonishchevDmitry/feedsd/pkg/url"
	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name        string
		status      int
		contentType string
		body        string
		text        string
		html        string
	}

	// FIXME(konishchev): Add JS test
	testCases := []testCase{{
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
	}}

	// Depending on content type Chrome may prepend the output with the following text:
	// "This XML file does not appear to have any style information associated with it. The document tree is shown below."
	for _, contentType := range rss.PossibleContentTypes {
		testCases = append(testCases, testCase{
			name:        contentType,
			status:      http.StatusOK,
			contentType: contentType,
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
		})
	}

	ctx, stop, err := Configure(testutil.Context(t))
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
			require.Equal(t, testCase.status, response.StatusCode)
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
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.7339.186 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36",
	} {
		t.Run(userAgent, func(t *testing.T) {
			matches := userAgentRe.FindStringSubmatch(userAgent)
			require.NotEmpty(t, matches)
			require.Equal(t, "Chrome", matches[2])
		})
	}
}
