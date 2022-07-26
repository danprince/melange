package main

import (
	"regexp"
	"strings"
)

var slugRegex = regexp.MustCompile(`[^\w]`)

func slugify(s string) string {
	s = slugRegex.ReplaceAllString(s, "_")
	s = strings.TrimSuffix(s, "_")
	s = strings.TrimPrefix(s, "_")
	return s
}
