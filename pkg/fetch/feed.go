package fetch

import (
	"context"
	"io"
	"net/url"

	"github.com/mmcdole/gofeed"

	"github.com/KonishchevDmitry/feedsd/pkg/rss"
)

func RSS(ctx context.Context, url *url.URL, options ...Option) (*rss.Feed, error) {
	return fetch(ctx, url, rss.PossibleContentTypes, rss.Read, options...)
}

func Feed(ctx context.Context, url *url.URL, options ...Option) (*gofeed.Feed, error) {
	contentTypes := append(
		[]string{"application/atom+xml"},
		rss.PossibleContentTypes...)

	return fetch(ctx, url, contentTypes, func(body io.Reader, ignoreCharset bool) (*gofeed.Feed, error) {
		// TODO(konishchev): gofeed doesn't allow to ignore charset, so we don't support this now
		return gofeed.NewParser().Parse(body)
	}, options...)
}
