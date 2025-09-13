package filter

import "strings"

const SectionDelimiter = " :: "

type Blacklist []string

func (b Blacklist) IsBlacklisted(category string) bool {
	for _, blacklisted := range b {
		if category == blacklisted || strings.HasPrefix(category, blacklisted+SectionDelimiter) {
			return true
		}
	}
	return false
}
