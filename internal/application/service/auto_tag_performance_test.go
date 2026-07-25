package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
)

const autoTagGenerationBaseline = 10 * time.Second

type autoTagBenchmarkChat struct {
	classification string
}

func (s *autoTagBenchmarkChat) Chat(
	_ context.Context,
	_ []chat.Message,
	options *chat.ChatOptions,
) (*types.ChatResponse, error) {
	if options != nil && options.Format != nil {
		return &types.ChatResponse{Content: s.classification}, nil
	}
	return &types.ChatResponse{Content: "Concise classification summary."}, nil
}

func (*autoTagBenchmarkChat) ChatStream(
	context.Context,
	[]chat.Message,
	*chat.ChatOptions,
) (<-chan types.StreamResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (*autoTagBenchmarkChat) GetModelName() string { return "auto-tag-benchmark" }
func (*autoTagBenchmarkChat) GetModelID() string   { return "auto-tag-benchmark" }

func autoTagPerformanceFixture() ([]*types.Chunk, map[string]autoTagDimensionOption, *autoTagBenchmarkChat) {
	const dimensionCount = 7
	const tagsPerDimension = 20

	chunks := make([]*types.Chunk, 1000)
	for i := range chunks {
		chunks[i] = &types.Chunk{
			Content:    strings.Repeat("document classification content ", 8),
			ChunkIndex: len(chunks) - i,
			StartAt:    len(chunks) - i,
			ChunkType:  types.ChunkTypeText,
		}
	}

	options := make(map[string]autoTagDimensionOption, dimensionCount)
	assignments := make([]string, 0, dimensionCount)
	for dimensionIndex := 0; dimensionIndex < dimensionCount; dimensionIndex++ {
		dimensionID := fmt.Sprintf("dimension-%d", dimensionIndex)
		tags := make(map[string]*types.TagDictionary, tagsPerDimension)
		for tagIndex := 0; tagIndex < tagsPerDimension; tagIndex++ {
			tagID := fmt.Sprintf("tag-%d-%d", dimensionIndex, tagIndex)
			tags[tagID] = &types.TagDictionary{
				ID:          tagID,
				DimensionID: dimensionID,
				Name:        fmt.Sprintf("Tag %d", tagIndex),
				SortOrder:   tagIndex,
			}
		}
		options[dimensionID] = autoTagDimensionOption{
			Dimension: &types.TagDimension{
				ID:          dimensionID,
				Name:        fmt.Sprintf("Dimension %d", dimensionIndex),
				Description: "Performance baseline dimension",
				SortOrder:   dimensionIndex,
			},
			Tags: tags,
		}
		assignments = append(assignments, fmt.Sprintf(
			`{"dimension_id":%q,"tag_id":%q,"confidence":0.95}`,
			dimensionID,
			fmt.Sprintf("tag-%d-0", dimensionIndex),
		))
	}

	return chunks, options, &autoTagBenchmarkChat{
		classification: `{"assignments":[` + strings.Join(assignments, ",") + `]}`,
	}
}

func runAutoTagPerformancePipeline(
	ctx context.Context,
	chunks []*types.Chunk,
	options map[string]autoTagDimensionOption,
	model chat.Chat,
) error {
	content := autoTagDocumentContent(chunks, autoTagMaxInputChars)
	summary, err := extractAutoTagSummary(ctx, model, content)
	if err != nil {
		return err
	}
	assignments, err := classifyAutoTags(ctx, model, summary, options)
	if err != nil {
		return err
	}
	_, err = validateAutoTagAssignments(types.AutoTagGenerationPayload{
		TenantID:    1,
		KnowledgeID: "performance-document",
	}, assignments, options)
	return err
}

func TestAutoTagGenerationPerformanceBaseline(t *testing.T) {
	chunks, options, model := autoTagPerformanceFixture()
	started := time.Now()
	if err := runAutoTagPerformancePipeline(context.Background(), chunks, options, model); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed >= autoTagGenerationBaseline {
		t.Fatalf("local auto-tag pipeline took %s, baseline is <%s", elapsed, autoTagGenerationBaseline)
	}
}

func BenchmarkAutoTagGeneration(b *testing.B) {
	chunks, options, model := autoTagPerformanceFixture()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := runAutoTagPerformancePipeline(ctx, chunks, options, model); err != nil {
			b.Fatal(err)
		}
	}
}
