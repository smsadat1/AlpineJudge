package assert

import (
	"slices"
	"testing"
)

// string assertion
func String(t *testing.T, expected string, achieved string) {

	t.Helper()
	if expected != achieved {
		t.Errorf("Expected %s, got %s", expected, achieved)
	}
}

func Slice(t *testing.T, expected []string, achieved []string) {
	if !slices.Equal(expected, achieved) {
		t.Errorf("Expected string slice doesn't match with achieved\n")
	}
}

// number assertion
func Uint32(t *testing.T, expected uint32, achieved uint32) {
	t.Helper()
	if expected != achieved {
		t.Errorf("Expected %d, got %d", expected, achieved)
	}
}

// boolean assertion
func Bool(t *testing.T, expected bool, achieved bool) {

	t.Helper()
	if expected != achieved {
		t.Errorf("Expected %v, got %v", expected, achieved)
	}
}
