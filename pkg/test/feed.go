package test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/KonishchevDmitry/feedsd/pkg/browser"
	"github.com/KonishchevDmitry/feedsd/pkg/feed"
	"github.com/KonishchevDmitry/feedsd/pkg/fetch"
	"github.com/KonishchevDmitry/feedsd/pkg/test/testutil"
)

func Feed(t *testing.T, generator feed.Feed, opts ...Option) {
	t.Parallel()

	var err error

	var options options
	for _, opt := range opts {
		opt(&options)
	}

	ctx := testutil.Context(t)
	ctx = fetch.WithContext(ctx, prometheus.NewHistogram(prometheus.HistogramOpts{}))

	if options.needsBrowser {
		var stop func()
		ctx, stop, err = browser.Configure(ctx)
		require.NoError(t, err)
		defer stop()
	}

	feed, err := generator.Get(ctx)
	require.NoError(t, err)

	if !options.mayBeEmpty {
		require.NotEmpty(t, feed.Items)
	}

	if !options.mayHaveEmptyDescription {
		for _, item := range feed.Items {
			require.NotEmpty(t, item.Description)
		}
	}
}

type options struct {
	mayBeEmpty              bool
	mayHaveEmptyDescription bool
	needsBrowser            bool
}

type Option func(o *options)

func MayBeEmpty() Option {
	return func(o *options) {
		o.mayBeEmpty = true
	}
}

func MayHaveEmptyDescription() Option {
	return func(o *options) {
		o.mayHaveEmptyDescription = true
	}
}

func NeedsBrowser() Option {
	return func(o *options) {
		o.needsBrowser = true
	}
}
