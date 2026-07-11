// Package testutil holds small helpers shared across *_test.go files
// so they are not copy-pasted into every package that needs them.
package testutil

import "strings"

// Contains reports whether substr is present in s. It replaces the
// previously duplicated per-package contains/containsStr helpers.
func Contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
