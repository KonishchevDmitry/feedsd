package fetch

import (
	"github.com/samber/mo"

	"github.com/KonishchevDmitry/feedsd/pkg/browser"
)

type Option func(o *options)

type options struct {
	emulateBrowser mo.Option[[]browser.QueryOption]
}

func EmulateBrowser(queryOptions ...browser.QueryOption) Option {
	return func(o *options) {
		o.emulateBrowser = mo.Some(queryOptions)
	}
}
