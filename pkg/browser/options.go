package browser

import (
	"net/url"

	"github.com/samber/mo"
)

type options struct {
	remote mo.Option[string]
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
