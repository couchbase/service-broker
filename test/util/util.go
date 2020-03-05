package util

import (
	"testing"
)

// Assert asserts a condition holds, causing test failure if it doesn't.
func Assert(t *testing.T, condition bool) {
	if !condition {
		t.Fatalf("assertion failed")
	}
}
