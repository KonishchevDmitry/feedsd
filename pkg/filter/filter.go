package filter

import "strings"

const sectionDelimiter = " :: "

func MakeCategory(sections ...string) string {
	return strings.Join(sections, sectionDelimiter)
}

type Blacklist []string

func (b Blacklist) IsBlacklisted(category string) bool {
	for _, blacklisted := range b {
		if category == blacklisted || strings.HasPrefix(category, blacklisted+sectionDelimiter) {
			return true
		}
	}
	return false
}
