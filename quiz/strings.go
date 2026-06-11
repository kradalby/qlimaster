package quiz

import "strings"

// equalFold is a small wrapper around strings.EqualFold kept here so the
// package has a single place for string-comparison helpers.
func equalFold(a, b string) bool {
	return strings.EqualFold(a, b)
}
