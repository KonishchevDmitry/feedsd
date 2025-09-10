package feed

import (
	"context"

	"github.com/KonishchevDmitry/feedsd/pkg/rss"
)

type Feed interface {
	Name() string
	Get(ctx context.Context) (*rss.Feed, error)
}
