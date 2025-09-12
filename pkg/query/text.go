package query

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// FIXME(konishchev): Implement
func Text(selection *goquery.Selection) string {
	return strings.TrimSpace(selection.Text())
	// return TrimText(selection.Text())
}
