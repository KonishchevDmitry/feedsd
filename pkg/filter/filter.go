package filter

import (
	"slices"
	"strings"
)

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

func (b Blacklist) HasBlacklisted(categories []string) bool {
	return slices.ContainsFunc(categories, b.IsBlacklisted)
}

type JointBlacklist [][]string

func (b JointBlacklist) IsBlacklisted(categories []string) bool {
BlacklistLoop:
	for _, blacklist := range b {
		for _, blacklisted := range blacklist {
			if !slices.Contains(categories, blacklisted) {
				continue BlacklistLoop
			}
		}

		if len(blacklist) != 0 {
			return true
		}
	}

	return false
}
