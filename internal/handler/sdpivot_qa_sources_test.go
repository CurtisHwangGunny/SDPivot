package handler

import (
	"encoding/json"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestBuildSDPivotQASourcesMatched(t *testing.T) {
	sources, status := buildSDPivotQASources([]types.SDPivotDocumentChunk{
		{DocumentID: "doc-1"},
		{DocumentID: "doc-1"},
		{DocumentID: "doc-2"},
	})
	if status != "matched" {
		t.Fatalf("status = %q, want matched", status)
	}
	var ids []string
	if err := json.Unmarshal([]byte(sources), &ids); err != nil {
		t.Fatalf("decode sources: %v", err)
	}
	if len(ids) != 2 || ids[0] != "doc-1" || ids[1] != "doc-2" {
		t.Fatalf("sources = %#v, want stable unique document IDs", ids)
	}
}

func TestBuildSDPivotQASourcesNoMatch(t *testing.T) {
	for _, chunks := range [][]types.SDPivotDocumentChunk{
		nil,
		{},
		{{DocumentID: ""}},
	} {
		sources, status := buildSDPivotQASources(chunks)
		if sources != "[]" || status != "no_match" {
			t.Fatalf("sources = %q status = %q, want [] and no_match", sources, status)
		}
	}
}
