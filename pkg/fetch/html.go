package fetch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/url"
	"strings"

	logging "github.com/KonishchevDmitry/go-easy-logging"
	"github.com/PuerkitoBio/goquery"
	"github.com/samber/mo"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"

	"github.com/KonishchevDmitry/feedsd/pkg/query"
)

func HTML(ctx context.Context, url *url.URL, options ...Option) (*goquery.Document, error) {
	return fetch(ctx, url, []string{"text/html"}, func(body io.Reader) (*goquery.Document, error) {
		data, err := io.ReadAll(body)
		if err != nil {
			return nil, err
		}

		doc, err := html.Parse(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}

		if encoding, ok := getHTMLCharset(ctx, doc, url); ok && encoding != "utf-8" && encoding != "utf8" {
			charsetReader, err := charset.NewReaderLabel(encoding, bytes.NewReader(data))
			if err != nil {
				return nil, fmt.Errorf("the document has an unknown charset encoding: %q", encoding)
			}

			data, err = io.ReadAll(charsetReader)
			if err != nil {
				return nil, fmt.Errorf("failed to decode the document using %s charset: %w", encoding, err)
			}

			doc, err = html.Parse(bytes.NewReader(data))
			if err != nil {
				return nil, err
			}
		}

		return goquery.NewDocumentFromNode(doc), nil
	}, options...)
}

func Description(
	ctx context.Context, url *url.URL, baseURL *url.URL, parser func(doc *goquery.Document) (*goquery.Selection, error),
	options ...Option,
) (_ string, retErr error) {
	doc, err := HTML(ctx, url, options...)
	if err != nil {
		return "", err
	}
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("failed to fetch description from %s: %w", url, err)
		}
	}()

	selection, err := parser(doc)
	if err != nil {
		return "", err
	}

	return query.Description(selection, baseURL)
}

func getHTMLCharset(ctx context.Context, doc *html.Node, uri *url.URL) (string, bool) {
	node, ok := findHTMLNode(doc, "html")
	if ok {
		node, ok = findHTMLNode(node, "head")
	}
	if !ok {
		return "", false
	}

	var (
		charset       mo.Option[string]
		isHTTPCharset = true
	)

	for node := node.FirstChild; node != nil; node = node.NextSibling {
		if node.Type != html.ElementNode || node.Data != "meta" {
			continue
		}

		attrs := make(map[string]string)
		for _, attr := range node.Attr {
			attrs[strings.ToLower(attr.Key)] = strings.ToLower(attr.Val)
		}

		if attrs["http-equiv"] == "content-type" {
			if _, params, err := mime.ParseMediaType(attrs["content"]); err != nil {
				logging.L(ctx).Warnf(
					`Got an invalid content type of %s from <meta http-equiv="Content-Type"> tag: %q.`,
					uri, attrs["content"])
			} else if encoding := params["charset"]; encoding != "" && isHTTPCharset {
				charset = mo.Some(encoding)
			}
		}

		if encoding := attrs["charset"]; encoding != "" {
			charset = mo.Some(encoding)
			isHTTPCharset = false
		}
	}

	return charset.Get()
}

func findHTMLNode(node *html.Node, name string) (*html.Node, bool) {
	for node = node.FirstChild; node != nil; node = node.NextSibling {
		if node.Type == html.ElementNode && node.Data == name {
			return node, true
		}
	}
	return nil, false
}
