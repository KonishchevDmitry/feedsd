package parse

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	spacesRe        = regexp.MustCompile(`\p{Z}+`)
	formatControlRe = regexp.MustCompile(`\p{Cf}+`)
)

func TrimText(text string) string {
	text = spacesRe.ReplaceAllString(text, " ")
	text = formatControlRe.ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}

var urlRe = regexp.MustCompile(`\bhttps?://[^\s(,!)]+`)

func TextToHTML(text string) string {
	html := html.EscapeString(text)

	html = urlRe.ReplaceAllStringFunc(html, func(match string) string {
		url := strings.TrimRight(match, "?.")
		punctuation := match[len(url):]
		return fmt.Sprintf(`<a href="%s">%s</a>%s`, url, url, punctuation)
	})

	return strings.ReplaceAll(html, "\n", "<br/>\n")
}
