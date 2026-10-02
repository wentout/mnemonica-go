package mnemonica

import (
	"errors"
	"strings"
	"testing"
)

// TestNewIDPrefixRandomFailure covers the crypto/rand failure fallback:
// ids only need per-process uniqueness, so a fixed prefix beats a crash.
// Internal (white-box) because the reader is an unexported package var.
func TestNewIDPrefixRandomFailure(t *testing.T) {
	original := processRandomRead
	processRandomRead = func([]byte) (int, error) {
		return 0, errors.New("no entropy")
	}
	t.Cleanup(func() {
		processRandomRead = original
	})

	prefix := newIDPrefix()
	if !strings.HasPrefix(prefix, "00000000") {
		t.Errorf("prefix after a random failure = %q, want the fixed fallback", prefix)
	}
}
