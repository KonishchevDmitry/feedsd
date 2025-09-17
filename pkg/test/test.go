package test

import (
	"context"
	"math/rand/v2"
	"testing"

	logging "github.com/KonishchevDmitry/go-easy-logging"
	"github.com/PuerkitoBio/goquery"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/KonishchevDmitry/feedsd/pkg/feed"
	"github.com/KonishchevDmitry/feedsd/pkg/fetch"
)

func Context(t *testing.T) context.Context {
	return logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())
}

func Feed(t *testing.T, generator feed.Feed, mayBeEmpty bool) {
	t.Parallel()

	ctx := Context(t)
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
