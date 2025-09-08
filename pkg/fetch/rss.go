package fetch

import (
	"context"
	"net/url"

	"github.com/KonishchevDmitry/newslib/pkg/rss"
)

func Feed(ctx context.Context, url *url.URL) (*rss.Feed, error) {
	return fetch(ctx, url, []string{"application/rss+xml", "application/xml", "text/xml"}, rss.Read)
}
