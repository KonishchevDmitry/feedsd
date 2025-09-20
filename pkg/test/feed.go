package test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/KonishchevDmitry/feedsd/pkg/feed"
	"github.com/KonishchevDmitry/feedsd/pkg/fetch"
	"github.com/KonishchevDmitry/feedsd/pkg/test/testutil"
)

func Feed(t *testing.T, generator feed.Feed, opts ...FeedOption) {
	t.Parallel()

	var options options
	for _, opt := range opts {
		opt(&options)
	}

	ctx := testutil.Context(t)
	ctx = fetch.WithContext(ctx, prometheus.NewHistogram(prometheus.HistogramOpts{}))

	feed, err := generator.Get(ctx)
	require.NoError(t, err)

	if !options.mayBeEmpty {
		require.NotEmpty(t, feed.Items)
	}

	for _, item := range feed.Items {
		require.NotEmpty(t, item.Description)
	}
}

type options struct {
	mayBeEmpty bool
}

type FeedOption func(o *options)

func MayBeEmpty() FeedOption {
	return func(o *options) {
		o.mayBeEmpty = true
	}
}
