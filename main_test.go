package main

import "testing"

func TestShouldIgnore(t *testing.T) {
	tests := map[string]bool{
		"_layout":     true,
		".git":        true,
		"markdown.md": false,
		"index.html":  false,
	}

	for name, expected := range tests {
		if shouldIgnore(name) != expected {
			t.Fatalf("expected %s to be: %t", name, expected)
		}
	}
}
