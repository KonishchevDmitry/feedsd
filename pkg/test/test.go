package test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/KonishchevDmitry/feedsd/pkg/feed"
	"github.com/KonishchevDmitry/feedsd/pkg/fetch"
	"github.com/KonishchevDmitry/feedsd/pkg/test/testutil"
)

func Feed(t *testing.T, generator feed.Feed, mayBeEmpty bool) {
	t.Parallel()

	ctx := testutil.Context(t)
	ctx = fetch.WithContext(ctx, prometheus.NewHistogram(prometheus.HistogramOpts{}))

	feed, err := generator.Get(ctx)
	require.NoError(t, err)

	if !mayBeEmpty {
		require.NotEmpty(t, feed.Items)
	}

	for _, item := range feed.Items {
		require.NotEmpty(t, item.Description)
	}
}
