package browser

import "context"

type configurationContext struct {
	userAgent string
}

type contextKey struct{}

func getConfigurationContext(ctx context.Context) (*configurationContext, bool) {
	configurationContext, ok := ctx.Value(contextKey{}).(*configurationContext)
	return configurationContext, ok
}
