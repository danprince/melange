package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"path/filepath"
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

func shouldIgnore(name string) bool {
	return name[0] == '_' || name[0] == '.'
}

func isPageFile(name string) bool {
	return filepath.Ext(name) == ".md"
}

func shortHash(s string) string {
	h := fnv.New32a()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum32())
}

func toJson(props *props) string {
	out, _ := json.Marshal(props)
	return string(out)
}
