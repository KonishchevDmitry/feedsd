package browser

import (
	"errors"
	"net/url"
	"time"

	"github.com/samber/mo"
)

type options struct {
	remote mo.Option[string]

	headful        bool
	persistentData mo.Option[string]
}

func getOptions(opts []Option) (options, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	local := o.headful || o.persistentData.IsPresent()
	if local && o.remote.IsPresent() {
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

func Headful() Option {
	return func(o *options) {
		o.headful = true
	}
}

func PersistentData(daemonName string) Option {
	return func(o *options) {
		o.persistentData = mo.Some(daemonName)
	}
}

type queryOptions struct {
	sleep time.Duration
}

type QueryOption func(o *queryOptions)

func Sleep(duration time.Duration) QueryOption {
	return func(o *queryOptions) {
		o.sleep = duration
	}
}
