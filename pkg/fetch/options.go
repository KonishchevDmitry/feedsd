package fetch

type Option func(o *options)

type options struct {
	emulateBrowser bool
}

func EmulateBrowser() Option {
	return func(o *options) {
		o.emulateBrowser = true
	}
}
