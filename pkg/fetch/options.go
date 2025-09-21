package fetch

import (
	"github.com/KonishchevDmitry/feedsd/pkg/browser"
	"github.com/samber/mo"
)

type Option func(o *options)

type options struct {
	emulateBrowser mo.Option[[]browser.Option]
}

func EmulateBrowser(opts ...browser.Option) Option {
	return func(o *options) {
		o.emulateBrowser = mo.Some(opts)
	}
}
