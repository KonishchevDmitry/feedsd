package url

import (
	"fmt"
	"net/url"
	"strings"
)

type URL = url.URL

func MustURL(value string) *url.URL {
	url, err := url.Parse(value)
	if err != nil {
		panic(fmt.Sprintf("Invalid URL: %s", value))
	}
	return url
}

func GetURL(base *url.URL, link string) (*url.URL, error) {
	if strings.HasPrefix(link, "/") {
		return base.JoinPath(link), nil
	}

	url, err := url.Parse(link)
	if err != nil {
		return nil, fmt.Errorf("got an invalid link: %q", link)
	}

	return url, nil
}
