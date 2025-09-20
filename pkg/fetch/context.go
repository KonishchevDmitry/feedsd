package fetch

import (
	"context"
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

type fetchContext struct {
	duration prometheus.Observer
}

type contextKey struct{}

func WithContext(ctx context.Context, duration prometheus.Observer) context.Context {
	return context.WithValue(ctx, contextKey{}, &fetchContext{
		duration: duration,
	})
}

func getContext(ctx context.Context) (*fetchContext, error) {
	context, ok := ctx.Value(contextKey{}).(*fetchContext)
	if !ok {
		return nil, errors.New("fetch context is missing")
	}
	return context, nil
}
