package testutil

import (
	"context"
	"math/rand/v2"
	"testing"

	logging "github.com/KonishchevDmitry/go-easy-logging"
	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap/zaptest"
)

func Context(t *testing.T) context.Context {
	return logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())
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
