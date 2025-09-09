package newslib

import (
	"context"
	"fmt"
	"net/url"

	"github.com/KonishchevDmitry/newslib/pkg/rss"
)

type FeedGenerator interface {
	Name() string
	Get(ctx context.Context) (*rss.Feed, error)
}

// FIXME(konishchev): Move out here?
func MustURL(value string) *url.URL {
	url, err := url.Parse(value)
	if err != nil {
		panic(fmt.Sprintf("Invalid URL: %s", value))
	}
	return url
}
