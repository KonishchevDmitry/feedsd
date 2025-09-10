package util

import (
	"fmt"
	"net/url"
)

func MustURL(value string) *url.URL {
	url, err := url.Parse(value)
	if err != nil {
		panic(fmt.Sprintf("Invalid URL: %s", value))
	}
	return url
}
