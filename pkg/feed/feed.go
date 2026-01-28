package feed

import (
	"context"

	"github.com/KonishchevDmitry/feedsd/pkg/rss"
)

type Feed interface {
	Name() string
	Get(ctx context.Context) (*rss.Feed, error)
}

type Params interface {
}

type ParametrizedFeed[P Params] interface {
	Name() string
	Path() (string, bool)
	Get(ctx context.Context, params P) (*rss.Feed, error)
}

func BindParams[P Params](feed ParametrizedFeed[P], params P) Feed {
	return &bindParamsAdapter[P]{
		feed:   feed,
		params: params,
	}
}

type bindParamsAdapter[P Params] struct {
	feed   ParametrizedFeed[P]
	params P
}

func (a *bindParamsAdapter[P]) Name() string {
	return a.feed.Name()
}

func (a *bindParamsAdapter[P]) Get(ctx context.Context) (*rss.Feed, error) {
	return a.feed.Get(ctx, a.params)
}
