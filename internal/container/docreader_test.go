package container

import (
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/config"
)

func TestInitDocReaderClientRequiresAddressInOPMode(t *testing.T) {
	t.Setenv("DOCREADER_ADDR", "")
	t.Setenv("DOCREADER_TRANSPORT", "")

	reader, err := initDocReaderClient(&config.Config{
		Product: &config.ProductConfig{OPMode: true},
	})
	if err == nil {
		t.Fatal("expected missing DOCREADER_ADDR to fail in OP mode")
	}
	if reader != nil {
		t.Fatal("expected no document reader when OP mode validation fails")
	}
	if !strings.Contains(err.Error(), "DOCREADER_ADDR is required in OP edition") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitDocReaderClientAllowsDisconnectedOutsideOPMode(t *testing.T) {
	t.Setenv("DOCREADER_ADDR", "")
	t.Setenv("DOCREADER_TRANSPORT", "")

	reader, err := initDocReaderClient(&config.Config{
		Product: &config.ProductConfig{OPMode: false},
	})
	if err != nil {
		t.Fatalf("expected non-OP mode to allow disconnected startup: %v", err)
	}
	if reader == nil {
		t.Fatal("expected a disconnected document reader outside OP mode")
	}
}
