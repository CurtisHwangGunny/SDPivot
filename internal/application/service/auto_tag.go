package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/Tencent/WeKnora/internal/utils"
	"github.com/hibiken/asynq"
)

const autoTagMaxInputChars = 24 * 1024

type autoTagGenerationService struct {
	knowledgeRepo interfaces.KnowledgeRepository
	kbService     interfaces.KnowledgeBaseService
	chunkService  interfaces.ChunkService
	modelService  interfaces.ModelService
	documentTags  interfaces.DocumentTagRepository
	spanTracker   SpanTracker
}

type autoTagAssignment struct {
	DimensionID string  `json:"dimension_id"`
	TagID       string  `json:"tag_id"`
	Confidence  float64 `json:"confidence"`
}
type autoTagResult struct {
	Assignments []autoTagAssignment `json:"assignments"`
}

func NewAutoTagGenerationService(knowledgeRepo interfaces.KnowledgeRepository, kbService interfaces.KnowledgeBaseService, chunkService interfaces.ChunkService, modelService interfaces.ModelService, documentTags interfaces.DocumentTagRepository, spanTracker SpanTracker) interfaces.TaskHandler {
	return &autoTagGenerationService{knowledgeRepo: knowledgeRepo, kbService: kbService, chunkService: chunkService, modelService: modelService, documentTags: documentTags, spanTracker: spanTracker}
}

func (s *autoTagGenerationService) Handle(ctx context.Context, task *asynq.Task) (retErr error) {
	var payload types.AutoTagGenerationPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal auto-tag payload: %w", err)
	}
	ctx = context.WithValue(ctx, types.TenantIDContextKey, payload.TenantID)
	if payload.Language != "" {
		ctx = context.WithValue(ctx, types.LanguageContextKey, payload.Language)
	}
	superseded := attemptSuperseded(ctx, s.spanTrackerOrNoop(), payload.KnowledgeID, payload.Attempt)
	defer func() {
		finalizeSubtaskDetached(ctx, s.knowledgeRepo, payload.KnowledgeID, "auto_tag", retErr, superseded, isFinalAsynqAttempt(ctx))
	}()
	if _, err := s.knowledgeRepo.GetKnowledgeByID(ctx, payload.TenantID, payload.KnowledgeID); err != nil {
		return err
	}
	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, payload.KnowledgeBaseID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(kb.SummaryModelID) == "" {
		return nil
	}
	dimensions, dictionary, err := s.documentTags.ListClassificationDictionary(ctx)
	if err != nil {
		return err
	}
	options := buildAutoTagOptions(dimensions, dictionary)
	if len(options) == 0 {
		return nil
	}
	chunks, err := s.chunkService.ListChunksByKnowledgeID(ctx, payload.KnowledgeID)
	if err != nil {
		return err
	}
	content := autoTagDocumentContent(chunks, autoTagMaxInputChars)
	if strings.TrimSpace(content) == "" {
		return nil
	}
	model, err := s.modelService.GetChatModel(ctx, kb.SummaryModelID)
	if err != nil {
		return err
	}
	summary, err := extractAutoTagSummary(ctx, model, content)
	if err != nil {
		return err
	}
	assignments, err := classifyAutoTags(ctx, model, summary, options)
	if err != nil {
		return err
	}
	rows, err := validateAutoTagAssignments(payload, assignments, options)
	if err != nil {
		return err
	}
	return s.documentTags.ReplaceDocumentTags(ctx, payload.TenantID, payload.KnowledgeID, rows)
}

func (s *autoTagGenerationService) spanTrackerOrNoop() SpanTracker {
	if s.spanTracker != nil {
		return s.spanTracker
	}
	return noopSpanTracker{}
}

type autoTagDimensionOption struct {
	Dimension *types.TagDimension
	Tags      map[string]*types.TagDictionary
}

func buildAutoTagOptions(dimensions []*types.TagDimension, dictionary []*types.TagDictionary) map[string]autoTagDimensionOption {
	options := make(map[string]autoTagDimensionOption)
	for _, d := range dimensions {
		if d != nil {
			options[d.ID] = autoTagDimensionOption{Dimension: d, Tags: map[string]*types.TagDictionary{}}
		}
	}
	for _, tag := range dictionary {
		if tag == nil {
			continue
		}
		option, ok := options[tag.DimensionID]
		if ok {
			option.Tags[tag.ID] = tag
			options[tag.DimensionID] = option
		}
	}
	for id, option := range options {
		if len(option.Tags) == 0 {
			delete(options, id)
		}
	}
	return options
}

func autoTagDocumentContent(chunks []*types.Chunk, maxChars int) string {
	filtered := make([]*types.Chunk, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk != nil && (chunk.ChunkType == types.ChunkTypeText || chunk.ChunkType == types.ChunkTypeImageOCR || chunk.ChunkType == types.ChunkTypeImageCaption) && strings.TrimSpace(chunk.Content) != "" {
			filtered = append(filtered, chunk)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].StartAt == filtered[j].StartAt {
			return filtered[i].ChunkIndex < filtered[j].ChunkIndex
		}
		return filtered[i].StartAt < filtered[j].StartAt
	})
	parts := make([]string, 0, len(filtered))
	for _, chunk := range filtered {
		parts = append(parts, strings.TrimSpace(chunk.Content))
	}
	return sampleLongContent(strings.Join(parts, "\n\n"), maxChars)
}

func extractAutoTagSummary(ctx context.Context, model chat.Chat, content string) (string, error) {
	thinking := false
	response, err := model.Chat(ctx, []chat.Message{{Role: "system", Content: "Summarize the supplied document for classification. Preserve its subject, organization, business domain, document type, sensitivity clues, and lifecycle state. Use only facts present. Output concise plain text."}, {Role: "user", Content: content}}, &chat.ChatOptions{Temperature: 0.1, MaxTokens: 1200, Thinking: &thinking})
	if err != nil {
		return "", fmt.Errorf("extract auto-tag summary: %w", err)
	}
	if response == nil || strings.TrimSpace(response.Content) == "" {
		return "", fmt.Errorf("extract auto-tag summary: empty response")
	}
	return strings.TrimSpace(response.Content), nil
}

func classifyAutoTags(ctx context.Context, model chat.Chat, summary string, options map[string]autoTagDimensionOption) ([]autoTagAssignment, error) {
	var prompt strings.Builder
	prompt.WriteString("Classify using only the IDs below. Return exactly one tag for every dimension. Confidence must be between 0 and 1.\n\n")
	ids := make([]string, 0, len(options))
	for id := range options {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		option := options[id]
		fmt.Fprintf(&prompt, "Dimension: %s (%s), id=%s\n", option.Dimension.Name, option.Dimension.Description, id)
		tagIDs := make([]string, 0, len(option.Tags))
		for tagID := range option.Tags {
			tagIDs = append(tagIDs, tagID)
		}
		sort.Strings(tagIDs)
		for _, tagID := range tagIDs {
			fmt.Fprintf(&prompt, "- %s: %s\n", tagID, option.Tags[tagID].Name)
		}
	}
	prompt.WriteString("\nDocument summary:\n" + summary)
	thinking := false
	response, err := model.Chat(ctx, []chat.Message{{Role: "system", Content: "Output only valid JSON matching the requested schema."}, {Role: "user", Content: prompt.String()}}, &chat.ChatOptions{Temperature: 0, MaxTokens: 1800, Thinking: &thinking, Format: utils.GenerateSchema[autoTagResult]()})
	if err != nil {
		return nil, fmt.Errorf("classify auto-tags: %w", err)
	}
	if response == nil {
		return nil, fmt.Errorf("classify auto-tags: empty response")
	}
	var result autoTagResult
	if err := json.Unmarshal([]byte(cleanLLMJSON(response.Content)), &result); err != nil {
		return nil, fmt.Errorf("parse auto-tag response: %w", err)
	}
	return result.Assignments, nil
}

func validateAutoTagAssignments(payload types.AutoTagGenerationPayload, assignments []autoTagAssignment, options map[string]autoTagDimensionOption) ([]*types.DocumentTag, error) {
	if len(assignments) != len(options) {
		return nil, fmt.Errorf("auto-tag response has %d assignments, want %d", len(assignments), len(options))
	}
	seen := map[string]struct{}{}
	rows := make([]*types.DocumentTag, 0, len(assignments))
	for _, assignment := range assignments {
		option, ok := options[assignment.DimensionID]
		if !ok {
			return nil, fmt.Errorf("unknown dimension %q", assignment.DimensionID)
		}
		if _, ok := seen[assignment.DimensionID]; ok {
			return nil, fmt.Errorf("repeated dimension %q", assignment.DimensionID)
		}
		if _, ok := option.Tags[assignment.TagID]; !ok {
			return nil, fmt.Errorf("invalid tag %q", assignment.TagID)
		}
		if assignment.Confidence < 0 || assignment.Confidence > 1 {
			return nil, fmt.Errorf("confidence outside [0,1]")
		}
		seen[assignment.DimensionID] = struct{}{}
		rows = append(rows, &types.DocumentTag{TenantID: payload.TenantID, DocumentID: payload.KnowledgeID, TagID: assignment.TagID, DimensionID: assignment.DimensionID, Confidence: assignment.Confidence})
	}
	return rows, nil
}

var _ = logger.Infof
