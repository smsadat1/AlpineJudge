package assert

import (
	"slices"
	"testing"
)

// string assertion
func String(t *testing.T, expected string, achieved string) {
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
	if expected != achieved {
		t.Errorf("Expected %d, got %d", expected, achieved)
	}
}

func Uint64(t *testing.T, expected uint64, achieved uint64) {
	if expected != achieved {
		t.Errorf("Expected %d, got %d", expected, achieved)
	}
}

func Int64(t *testing.T, expected int64, achieved int64) {
	if expected != achieved {
		t.Errorf("Expected %d, got %d", expected, achieved)
	}
}

// boolean assertion
func Bool(t *testing.T, expected bool, achieved bool) {
	if expected != achieved {
		t.Errorf("Expected %v, got %v", expected, achieved)
	}
}
