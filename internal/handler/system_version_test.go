package handler

import "testing"

func TestVersionDefaultIsOPProductVersion(t *testing.T) {
	if Version != "1.0.0" {
		t.Fatalf("Version = %q, want 1.0.0", Version)
	}
}
