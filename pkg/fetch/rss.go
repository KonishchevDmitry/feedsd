package fetch

import (
	"context"
	"net/url"

	"github.com/KonishchevDmitry/feedsd/pkg/rss"
)

func Feed(ctx context.Context, url *url.URL, options ...Option) (*rss.Feed, error) {
	return fetch(ctx, url, rss.PossibleContentTypes, rss.Read, options...)
}
