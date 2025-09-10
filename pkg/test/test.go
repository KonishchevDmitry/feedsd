package test

import (
	"context"
	"testing"

	"github.com/KonishchevDmitry/feedsd/pkg/feed"
	"github.com/stretchr/testify/require"
)

// FIXME(konishchev): Implement
func Feed(t *testing.T, generator feed.Feed, mayBeEmpty bool) {
	t.Parallel()
	ctx := context.Background()

	feed, err := generator.Get(ctx)
	require.NoError(t, err)
	if !mayBeEmpty {
		require.NotEmpty(t, feed.Items)
	}

	for _, item := range feed.Items {
		require.NotEmpty(t, item.Description)
		// require.False(t, strings.HasPrefix(item.Description, descriptionErrorMarker), item.Description)
	}
}
