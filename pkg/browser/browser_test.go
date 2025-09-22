package browser

import (
	"context"
	"io"
	"mime"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
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

	testCases := []struct {
		name        string
		status      int
		contentType string
		body        string
		result      string
	}{{
		name:        "text",
		status:      http.StatusOK,
		contentType: "text/plain",
		body:        "Some & text",
		result:      "Some & text",
	}, {
		name:        "error",
		status:      http.StatusInternalServerError,
		contentType: "text/html",
		body:        "Some error",
		result:      "<html><head></head><body>Some error</body></html>",
	}, {
		name:        "html",
		status:      http.StatusOK,
		contentType: "text/html",
		body:        "<html><body>Some text</body></html>",
		result:      "<html><head></head><body>Some text</body></html>",
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
		result: heredoc.Doc(`
			<?xml version="1.0" encoding="UTF-8"?>
			<rss version="2.0">
				<channel>
					<title>Feed title</title>
					<link>http://example.com/</link>
					<description>Feed description</description>
				</channel>
			</rss>
		`),
	}, {
		name:        "xml",
		status:      http.StatusOK,
		contentType: "text/xml",
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
		// For text/xml Content-Type Chrome:
		// * Cuts `<?xml ...?>`
		// * Prepends the output with the following text: "This XML file does not appear to have any style information associated with it. The document tree is shown below."
		result: heredoc.Doc(`
			<rss version="2.0">
				<channel>
					<title>Feed title</title>
					<link>http://example.com/</link>
					<description>Feed description</description>
				</channel>
			</rss>
		`),
	}, {
		name:        "js",
		status:      http.StatusOK,
		contentType: "text/html",
		body: heredoc.Doc(`
			<html>
				<body onload="changeText()">
					Initial Text
					<script>
						function changeText() {
							document.body.innerText = "Changed text";
						}
					</script>
				</body>
			</html>
		`),
		result: `<html><head></head><body onload="changeText()">Changed text</body></html>`,
	}, {
		name:        "bot-detection",
		status:      http.StatusOK,
		contentType: "text/html",
		body: heredoc.Doc(`
			<html>
				<body onload="botDetection()">
					<script>
						function botDetection() {
							if(navigator.webdriver) {
								document.body.innerText = "Bot is detected";
							} else {
								document.body.innerText = "Bot is not detected";
							}
						}
					</script>
				</body>
			</html>
		`),
		result: `<html><head></head><body onload="botDetection()">Bot is not detected</body></html>`,
	}}

	ctx, stop, err := Configure(testutil.Context(t))
	require.NoError(t, err)
	defer stop()

	var waitGroup sync.WaitGroup
	defer waitGroup.Wait()

	run := func(t *testing.T, name string, test func(t *testing.T)) {
		waitGroup.Go(func() {
			t.Run(name, test)
		})
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

			mediaType, _, err := mime.ParseMediaType(testCase.contentType)
			require.NoError(t, err)

			body, expected := response.Body, testCase.result
			if mediaType != "text/plain" {
				indentationRe := regexp.MustCompile(`(?m:^\s+|\s+$)`)
				body = indentationRe.ReplaceAllString(body, "")
				expected = indentationRe.ReplaceAllString(expected, "")
			}

			require.Equal(t, server.URL+"/", response.URL)
			require.Equal(t, testCase.status, response.StatusCode)
			require.Equal(t, testCase.contentType, response.ContentType)
			require.Equal(t, expected, body)
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
