package main

import (
	"regexp"
	"strings"
)

var slugRegex = regexp.MustCompile(`[^\w]`)

func slugify(s string) string {
	s = slugRegex.ReplaceAllString(s, "-")
	s = strings.TrimSuffix(s, "-")
	s = strings.TrimPrefix(s, "-")
	return s
}
