package fetch

import (
	"context"
	"net/url"

	"github.com/KonishchevDmitry/feedsd/pkg/rss"
)

func Feed(ctx context.Context, url *url.URL) (*rss.Feed, error) {
	return fetch(ctx, url, []string{rss.ContentType, "application/xml", "text/xml"}, rss.Read)
}
