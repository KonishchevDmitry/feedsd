package test

import (
	"context"
	"math/rand/v2"
	"testing"

	logging "github.com/KonishchevDmitry/go-easy-logging"
	"github.com/PuerkitoBio/goquery"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/KonishchevDmitry/feedsd/pkg/feed"
	"github.com/KonishchevDmitry/feedsd/pkg/fetch"
)

func Feed(t *testing.T, generator feed.Feed, mayBeEmpty bool) {
	t.Parallel()

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, zap.NewNop().Sugar())
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

func Limit[T any](items []T, limit int) []T {
	limit = min(limit, len(items))

	for count := range limit {
		selection := items[count:]

		index := rand.N(len(selection))
		item := selection[index]
		selection[index] = selection[0]

		items[count] = item
	}

	clear(items[limit:])
	return items[:limit]
}

func LimitSelection(selection *goquery.Selection, limit int) *goquery.Selection {
	selection.Nodes = Limit(selection.Nodes, limit)
	return selection
}
