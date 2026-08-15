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
	sources, status := buildSDPivotQASources([]sdpivotQAChunk{
		{SDPivotDocumentChunk: types.SDPivotDocumentChunk{DocumentID: "doc-1"}, DocumentTitle: "文档一", SpaceID: "space-1", SpaceName: "空间一"},
		{SDPivotDocumentChunk: types.SDPivotDocumentChunk{DocumentID: "doc-1"}, DocumentTitle: "文档一", SpaceID: "space-1", SpaceName: "空间一"},
		{SDPivotDocumentChunk: types.SDPivotDocumentChunk{DocumentID: "doc-2"}, DocumentTitle: "文档二", SpaceID: "space-2", SpaceName: "空间二"},
	})
	if status != "matched" {
		t.Fatalf("status = %q, want matched", status)
	}
	var decoded []sdpivotQASource
	if err := json.Unmarshal([]byte(sources), &decoded); err != nil {
		t.Fatalf("decode sources: %v", err)
	}
	if len(decoded) != 2 || decoded[0].DocumentID != "doc-1" || decoded[0].SpaceID != "space-1" || decoded[0].SpaceName != "空间一" || decoded[1].DocumentID != "doc-2" || decoded[1].SpaceID != "space-2" || decoded[1].SpaceName != "空间二" {
		t.Fatalf("sources = %#v, want stable unique documents with space metadata", decoded)
	}
}

func TestBuildSDPivotQASourcesNoMatch(t *testing.T) {
	for _, chunks := range [][]sdpivotQAChunk{
		nil,
		{},
		{{SDPivotDocumentChunk: types.SDPivotDocumentChunk{DocumentID: ""}}},
	} {
		sources, status := buildSDPivotQASources(chunks)
		if sources != "[]" || status != "no_match" {
			t.Fatalf("sources = %q status = %q, want [] and no_match", sources, status)
		}
	}
}

func TestResolveQASpaceIDs(t *testing.T) {
	tests := []struct {
		name           string
		spaceIDs       []string
		spaceID        string
		sessionSpaceID string
		want           []string
	}{
		{name: "new field wins", spaceIDs: []string{" space-1 ", "space-2", "space-1"}, spaceID: "legacy", sessionSpaceID: "session", want: []string{"space-1", "space-2"}},
		{name: "explicit empty means all", spaceIDs: []string{}, spaceID: "legacy", sessionSpaceID: "session", want: []string{}},
		{name: "legacy request field", spaceID: "legacy", sessionSpaceID: "session", want: []string{"legacy"}},
		{name: "session fallback", sessionSpaceID: "session", want: []string{"session"}},
		{name: "no scope means all", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveQASpaceIDs(tt.spaceIDs, tt.spaceID, tt.sessionSpaceID)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("resolveQASpaceIDs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
