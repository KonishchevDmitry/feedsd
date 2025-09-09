package feed

import (
	"context"

	"github.com/KonishchevDmitry/newslib/pkg/rss"
)

type Feed interface {
	Name() string
	Get(ctx context.Context) (*rss.Feed, error)
}
