package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type autoTagChatStub struct {
	responses []string
	calls     int
	options   []*chat.ChatOptions
}

func (s *autoTagChatStub) Chat(
	_ context.Context,
	_ []chat.Message,
	options *chat.ChatOptions,
) (*types.ChatResponse, error) {
	if s.calls >= len(s.responses) {
		return nil, errors.New("unexpected chat call")
	}
	s.options = append(s.options, options)
	response := &types.ChatResponse{Content: s.responses[s.calls]}
	s.calls++
	return response, nil
}

func (*autoTagChatStub) ChatStream(context.Context, []chat.Message, *chat.ChatOptions) (<-chan types.StreamResponse, error) {
	return nil, errors.New("not implemented")
}

func (*autoTagChatStub) GetModelName() string { return "auto-tag-stub" }
func (*autoTagChatStub) GetModelID() string   { return "auto-tag-stub" }

func TestAutoTagDocumentContent_SortsFiltersAndSamples(t *testing.T) {
	content := autoTagDocumentContent([]*types.Chunk{
		{Content: "ignored", ChunkType: types.ChunkTypeSummary, StartAt: 0},
		{Content: "second", ChunkType: types.ChunkTypeImageOCR, StartAt: 10},
		{Content: "first", ChunkType: types.ChunkTypeText, StartAt: 0},
	}, 100)
	assert.Equal(t, "first\n\nsecond", content)
}

func TestClassifyAutoTags_UsesSchemaAndParsesFencedJSON(t *testing.T) {
	dimensionID, tagID := "dimension-1", "tag-1"
	model := &autoTagChatStub{responses: []string{
		"```json\n{\"assignments\":[{\"dimension_id\":\"dimension-1\",\"tag_id\":\"tag-1\",\"confidence\":0.93}]}\n```",
	}}
	options := map[string]autoTagDimensionOption{
		dimensionID: {
			Dimension: &types.TagDimension{ID: dimensionID, Name: "Topic", SortOrder: 10},
			Tags: map[string]*types.TagDictionary{
				tagID: {ID: tagID, DimensionID: dimensionID, Name: "Engineering"},
			},
		},
	}

	assignments, err := classifyAutoTags(context.Background(), model, "summary", options)
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	assert.Equal(t, tagID, assignments[0].TagID)
	require.Len(t, model.options, 1)
	assert.NotEmpty(t, model.options[0].Format)
}

func TestValidateAutoTagAssignments_RejectsInvalidTagAndConfidence(t *testing.T) {
	dimensionID, tagID := "dimension-1", "tag-1"
	options := map[string]autoTagDimensionOption{
		dimensionID: {
			Dimension: &types.TagDimension{ID: dimensionID},
			Tags: map[string]*types.TagDictionary{
				tagID: {ID: tagID, DimensionID: dimensionID},
			},
		},
	}
	payload := types.AutoTagGenerationPayload{TenantID: 3, KnowledgeID: "doc-1"}

	_, err := validateAutoTagAssignments(payload, []autoTagAssignment{
		{DimensionID: dimensionID, TagID: "unknown", Confidence: 0.8},
	}, options)
	require.Error(t, err)

	_, err = validateAutoTagAssignments(payload, []autoTagAssignment{
		{DimensionID: dimensionID, TagID: tagID, Confidence: 1.1},
	}, options)
	require.Error(t, err)
}

func TestValidateAutoTagAssignments_BuildsRows(t *testing.T) {
	dimensionID, tagID := "dimension-1", "tag-1"
	options := map[string]autoTagDimensionOption{
		dimensionID: {
			Dimension: &types.TagDimension{ID: dimensionID},
			Tags: map[string]*types.TagDictionary{
				tagID: {ID: tagID, DimensionID: dimensionID},
			},
		},
	}
	payload := types.AutoTagGenerationPayload{TenantID: 3, KnowledgeID: "doc-1"}

	rows, err := validateAutoTagAssignments(payload, []autoTagAssignment{
		{DimensionID: dimensionID, TagID: tagID, Confidence: 0.88},
	}, options)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, uint64(3), rows[0].TenantID)
	assert.Equal(t, "doc-1", rows[0].DocumentID)
	assert.InDelta(t, 0.88, rows[0].Confidence, 0.0001)
}
