package main

import "testing"

func TestVersionFlagAndDefault(t *testing.T) {
	if opVersion != "1.0.0" {
		t.Fatalf("opVersion = %q, want 1.0.0", opVersion)
	}
	if !hasVersionFlag([]string{"--version"}) || !hasVersionFlag([]string{"-v"}) {
		t.Fatal("version flags were not recognized")
	}
}
