package query

import (
	"github.com/PuerkitoBio/goquery"

	"github.com/KonishchevDmitry/feedsd/pkg/parse"
)

func Text(selection *goquery.Selection) string {
	return parse.TrimText(selection.Text())
}
