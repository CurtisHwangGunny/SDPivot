package handler

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestExtractQAKeywordsPreservesMeaningfulChinesePhrase(t *testing.T) {
	keywords := extractQAKeywords("客户成功经营包含哪些内容？")
	if !slices.Contains(keywords, "客户成功经营") {
		t.Fatalf("expected customer-success phrase, got %v", keywords)
	}
	if slices.Contains(keywords, "包含") || slices.Contains(keywords, "哪些") {
		t.Fatalf("question words should be removed, got %v", keywords)
	}
}

func TestExtractQAKeywordsFallsBackToMeaningfulBigrams(t *testing.T) {
	keywords := extractQAKeywords("请根据知识库回答")
	if !slices.Contains(keywords, "知识") {
		t.Fatalf("expected meaningful bigram, got %v", keywords)
	}
	if slices.Contains(keywords, "根据") || slices.Contains(keywords, "回答") {
		t.Fatalf("stop bigrams should be removed, got %v", keywords)
	}
}

func TestExtractQAKeywordsEscapesAndBoundsTerms(t *testing.T) {
	keywords := extractQAKeywords("one two three four five six seven eight nine 100%_match")
	if len(keywords) != 8 {
		t.Fatalf("expected bounded keyword count, got %d: %v", len(keywords), keywords)
	}
	if escaped := escapeQAQuery(`100%_match\\`); escaped != `100\%\_match\\\\` {
		t.Fatalf("unexpected escaped query: %q", escaped)
	}
}

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
