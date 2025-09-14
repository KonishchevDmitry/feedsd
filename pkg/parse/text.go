package parse

import (
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
