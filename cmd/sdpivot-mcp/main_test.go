package main

import "testing"

func TestOPVersionDefault(t *testing.T) {
	if opVersion != "1.0.0" {
		t.Fatalf("opVersion = %q, want 1.0.0", opVersion)
	}
}
