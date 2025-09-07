package test

import (
	"context"
	"testing"

	"github.com/KonishchevDmitry/newslib/pkg/rss"
	"github.com/stretchr/testify/require"
)

// FIXME(konishchev): Implement
func Feed(t *testing.T, generator func(ctx context.Context) (*rss.Feed, error), mayBeEmpty bool) {
	t.Parallel()
	ctx := context.Background()

	feed, err := generator(ctx)
	require.NoError(t, err)
	if !mayBeEmpty {
		require.NotEmpty(t, feed.Items)
	}

	for _, item := range feed.Items {
		require.NotEmpty(t, item.Description)
		// require.False(t, strings.HasPrefix(item.Description, descriptionErrorMarker), item.Description)
	}
}
