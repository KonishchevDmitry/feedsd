package query

import (
	"errors"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"

	"github.com/KonishchevDmitry/feedsd/pkg/url"
)

func HTMLOrError(selection *goquery.Selection) string {
	html, err := selection.Html()
	if err != nil {
		html = fmt.Sprintf("[Failed to render the HTML: %s]", err)
	}
	return html
}

func Classes(selection *goquery.Selection) ([]string, error) {
	if len(selection.Nodes) != 1 || selection.Nodes[0].Type != html.ElementNode {
		return nil, errors.New("invalid selection to obtain node classes")
	}

	var classes []string

	for _, attr := range selection.Nodes[0].Attr {
		if attr.Key == "class" {
			for _, class := range strings.Split(attr.Val, " ") {
				if class != "" {
					classes = append(classes, class)
				}
			}
		}
	}

	return classes, nil
}

func Description(selection *goquery.Selection, baseURL *url.URL) (string, error) {
	selection = selection.Clone()
	selection.Find("script").Remove()

	if err := ForEach(selection.Find("a"), func(link *goquery.Selection) error {
		if href, ok := link.Attr("href"); ok && href != "" {
			href, err := url.Get(baseURL, href)
			if err != nil {
				return err
			}
			link.SetAttr("href", href.String())
		}
		return nil
	}); err != nil {
		return "", err
	}

	if err := ForEach(selection.Find("img"), func(image *goquery.Selection) error {
		if src, ok := image.Attr("src"); ok && src != "" {
			src, err := url.Get(baseURL, src)
			if err != nil {
				return err
			}
			image.SetAttr("src", src.String())
		}
		return nil
	}); err != nil {
		return "", err
	}

	return selection.Html()
}
