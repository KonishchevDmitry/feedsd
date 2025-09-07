package fetch

import (
	"context"

	"github.com/KonishchevDmitry/newslib/pkg/rss"
)

func Feed(ctx context.Context, url string) (*rss.Feed, error) {
	return fetch(ctx, url, []string{"application/rss+xml", "application/xml", "text/xml"}, rss.Read)
}
