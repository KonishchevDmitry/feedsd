package browser

import (
	"errors"
	"net/url"

	"github.com/samber/mo"
)

type options struct {
	remote     mo.Option[string]
	persistent mo.Option[string]
}

func getOptions(opts []Option) (options, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	if o.remote.IsPresent() && o.persistent.IsPresent() {
		return o, errors.New("mixed remote and local browser options")
	}

	return o, nil
}

type Option func(o *options)

func Remote(hostPort string) Option {
	return func(o *options) {
		url := url.URL{
			Scheme: "ws",
			Host:   hostPort,
		}
		o.remote = mo.Some(url.String())
	}
}

func Persistent(daemonName string) Option {
	return func(o *options) {
		o.persistent = mo.Some(daemonName)
	}
}
