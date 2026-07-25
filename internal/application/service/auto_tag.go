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
	DimensionID string  `json:"dimension_id" jsonschema:"ID of the supplied dimension"`
	TagID       string  `json:"tag_id" jsonschema:"ID of one supplied dictionary tag in that dimension"`
	Confidence  float64 `json:"confidence" jsonschema:"Confidence from 0 to 1"`
}

type autoTagResult struct {
	Assignments []autoTagAssignment `json:"assignments" jsonschema:"One assignment for each supplied dimension"`
}

func NewAutoTagGenerationService(
	knowledgeRepo interfaces.KnowledgeRepository,
	kbService interfaces.KnowledgeBaseService,
	chunkService interfaces.ChunkService,
	modelService interfaces.ModelService,
	documentTags interfaces.DocumentTagRepository,
	spanTracker SpanTracker,
) interfaces.TaskHandler {
	return &autoTagGenerationService{
		knowledgeRepo: knowledgeRepo,
		kbService:     kbService,
		chunkService:  chunkService,
		modelService:  modelService,
		documentTags:  documentTags,
		spanTracker:   spanTracker,
	}
}

func (s *autoTagGenerationService) tracker() SpanTracker {
	if s.spanTracker == nil {
		return noopSpanTracker{}
	}
	return s.spanTracker
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

	superseded := attemptSuperseded(ctx, s.tracker(), payload.KnowledgeID, payload.Attempt)
	if superseded {
		logger.Infof(ctx, "auto-tag: attempt %d superseded for %s", payload.Attempt, payload.KnowledgeID)
		return nil
	}
	defer func() {
		finalizeSubtaskDetached(ctx, s.knowledgeRepo, payload.KnowledgeID, "auto_tag",
			retErr, superseded, isFinalAsynqAttempt(ctx))
	}()

	span := beginAutoTagSpan(ctx, s.tracker(), payload)
	var spanErr error
	spanOut := types.JSONMap{}
	defer func() {
		if span == nil {
			return
		}
		if spanErr != nil {
			s.tracker().FailSpan(ctx, span, "AUTO_TAG_FAILED", spanErr.Error(), spanErr)
			return
		}
		s.tracker().EndSpan(ctx, span, spanOut)
	}()

	knowledge, err := s.knowledgeRepo.GetKnowledgeByID(ctx, payload.TenantID, payload.KnowledgeID)
	if err != nil {
		spanErr = err
		return fmt.Errorf("get knowledge for auto-tag: %w", err)
	}
	if knowledge.ParseStatus == types.ParseStatusCancelled || knowledge.ParseStatus == types.ParseStatusDeleting {
		spanOut["skipped"] = "knowledge_" + knowledge.ParseStatus
		return nil
	}

	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, payload.KnowledgeBaseID)
	if err != nil {
		spanErr = err
		return fmt.Errorf("get knowledge base for auto-tag: %w", err)
	}
	if strings.TrimSpace(kb.SummaryModelID) == "" {
		spanOut["skipped"] = "no_summary_model"
		return nil
	}
	spanOut["model_id"] = kb.SummaryModelID

	dimensions, dictionary, err := s.documentTags.ListClassificationDictionary(ctx)
	if err != nil {
		spanErr = err
		return fmt.Errorf("list classification dictionary: %w", err)
	}
	options := buildAutoTagOptions(dimensions, dictionary)
	if len(options) == 0 {
		spanOut["skipped"] = "empty_dictionary"
		return nil
	}
	spanOut["dimensions"] = len(options)

	chunks, err := s.chunkService.ListChunksByKnowledgeID(ctx, payload.KnowledgeID)
	if err != nil {
		spanErr = err
		return fmt.Errorf("list chunks for auto-tag: %w", err)
	}
	content := autoTagDocumentContent(chunks, autoTagMaxInputChars)
	if strings.TrimSpace(content) == "" {
		spanOut["skipped"] = "no_text_content"
		return nil
	}

	model, err := s.modelService.GetChatModel(ctx, kb.SummaryModelID)
	if err != nil {
		spanErr = err
		return fmt.Errorf("get auto-tag model: %w", err)
	}
	summary, err := extractAutoTagSummary(ctx, model, content)
	if err != nil {
		spanErr = err
		return err
	}
	spanOut["summary_chars"] = len([]rune(summary))

	assignments, err := classifyAutoTags(ctx, model, summary, options)
	if err != nil {
		spanErr = err
		return err
	}
	rows, err := validateAutoTagAssignments(payload, assignments, options)
	if err != nil {
		spanErr = err
		return err
	}
	if err := s.documentTags.ReplaceDocumentTags(ctx, payload.TenantID, payload.KnowledgeID, rows); err != nil {
		spanErr = err
		return fmt.Errorf("replace document tags: %w", err)
	}
	spanOut["assignments"] = len(rows)
	return nil
}

func beginAutoTagSpan(ctx context.Context, tracker SpanTracker, payload types.AutoTagGenerationPayload) *Span {
	if payload.Attempt <= 0 {
		return nil
	}
	parent := tracker.LookupStage(ctx, payload.KnowledgeID, payload.Attempt, types.StagePostProcess)
	if parent == nil {
		return nil
	}
	return tracker.BeginSubSpan(ctx, parent, "postprocess.auto_tag", types.SpanKindSubSpan, nil)
}

type autoTagDimensionOption struct {
	Dimension *types.TagDimension
	Tags      map[string]*types.TagDictionary
}

func buildAutoTagOptions(
	dimensions []*types.TagDimension,
	dictionary []*types.TagDictionary,
) map[string]autoTagDimensionOption {
	options := make(map[string]autoTagDimensionOption, len(dimensions))
	for _, dimension := range dimensions {
		if dimension != nil {
			options[dimension.ID] = autoTagDimensionOption{Dimension: dimension, Tags: map[string]*types.TagDictionary{}}
		}
	}
	for _, tag := range dictionary {
		if tag == nil {
			continue
		}
		option, ok := options[tag.DimensionID]
		if !ok {
			continue
		}
		option.Tags[tag.ID] = tag
		options[tag.DimensionID] = option
	}
	for id, option := range options {
		if len(option.Tags) == 0 {
			delete(options, id)
		}
	}
	return options
}

func autoTagDocumentContent(chunks []*types.Chunk, maxChars int) string {
	textChunks := make([]*types.Chunk, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk == nil {
			continue
		}
		switch chunk.ChunkType {
		case types.ChunkTypeText, types.ChunkTypeImageOCR, types.ChunkTypeImageCaption:
			if strings.TrimSpace(chunk.Content) != "" {
				textChunks = append(textChunks, chunk)
			}
		}
	}
	sort.SliceStable(textChunks, func(i, j int) bool {
		if textChunks[i].StartAt == textChunks[j].StartAt {
			return textChunks[i].ChunkIndex < textChunks[j].ChunkIndex
		}
		return textChunks[i].StartAt < textChunks[j].StartAt
	})
	parts := make([]string, 0, len(textChunks))
	for _, chunk := range textChunks {
		parts = append(parts, strings.TrimSpace(chunk.Content))
	}
	return sampleLongContent(strings.Join(parts, "\n\n"), maxChars)
}

func extractAutoTagSummary(ctx context.Context, model chat.Chat, content string) (string, error) {
	thinking := false
	response, err := model.Chat(ctx, []chat.Message{
		{Role: "system", Content: "Summarize the supplied document for classification. Preserve its subject, owning organization, business domain, named projects, document type, sensitivity clues, and lifecycle state. Use only facts present in the document. Output a concise plain-text summary in the document's language."},
		{Role: "user", Content: content},
	}, &chat.ChatOptions{Temperature: 0.1, MaxTokens: 1200, Thinking: &thinking})
	if err != nil {
		return "", fmt.Errorf("extract auto-tag summary: %w", err)
	}
	if response == nil {
		return "", fmt.Errorf("extract auto-tag summary: empty response")
	}
	summary := strings.TrimSpace(response.Content)
	if summary == "" {
		return "", fmt.Errorf("extract auto-tag summary: empty response")
	}
	return summary, nil
}

func classifyAutoTags(
	ctx context.Context,
	model chat.Chat,
	summary string,
	options map[string]autoTagDimensionOption,
) ([]autoTagAssignment, error) {
	var prompt strings.Builder
	prompt.WriteString("Classify the document summary using only the IDs below. Return exactly one tag for every listed dimension. Never invent an ID. Confidence must be between 0 and 1.\n\n")
	ids := make([]string, 0, len(options))
	for id := range options {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return options[ids[i]].Dimension.SortOrder < options[ids[j]].Dimension.SortOrder
	})
	for _, id := range ids {
		option := options[id]
		fmt.Fprintf(&prompt, "Dimension: %s (%s), id=%s\n", option.Dimension.Name, option.Dimension.Description, id)
		tagIDs := make([]string, 0, len(option.Tags))
		for tagID := range option.Tags {
			tagIDs = append(tagIDs, tagID)
		}
		sort.Slice(tagIDs, func(i, j int) bool {
			left, right := option.Tags[tagIDs[i]], option.Tags[tagIDs[j]]
			if left.SortOrder == right.SortOrder {
				return left.Name < right.Name
			}
			return left.SortOrder < right.SortOrder
		})
		for _, tagID := range tagIDs {
			fmt.Fprintf(&prompt, "- %s: %s\n", tagID, option.Tags[tagID].Name)
		}
		prompt.WriteByte('\n')
	}
	prompt.WriteString("Document summary:\n")
	prompt.WriteString(summary)

	thinking := false
	response, err := model.Chat(ctx, []chat.Message{
		{Role: "system", Content: "You are a deterministic document classifier. Output only valid JSON matching the requested schema."},
		{Role: "user", Content: prompt.String()},
	}, &chat.ChatOptions{
		Temperature: 0,
		MaxTokens:   1800,
		Thinking:    &thinking,
		Format:      utils.GenerateSchema[autoTagResult](),
	})
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

func validateAutoTagAssignments(
	payload types.AutoTagGenerationPayload,
	assignments []autoTagAssignment,
	options map[string]autoTagDimensionOption,
) ([]*types.DocumentTag, error) {
	if len(assignments) != len(options) {
		return nil, fmt.Errorf("auto-tag response has %d assignments, want %d", len(assignments), len(options))
	}
	seen := make(map[string]struct{}, len(assignments))
	rows := make([]*types.DocumentTag, 0, len(assignments))
	for _, assignment := range assignments {
		option, ok := options[assignment.DimensionID]
		if !ok {
			return nil, fmt.Errorf("auto-tag response contains unknown dimension %q", assignment.DimensionID)
		}
		if _, duplicate := seen[assignment.DimensionID]; duplicate {
			return nil, fmt.Errorf("auto-tag response repeats dimension %q", assignment.DimensionID)
		}
		if _, ok := option.Tags[assignment.TagID]; !ok {
			return nil, fmt.Errorf("auto-tag response contains invalid tag %q for dimension %q", assignment.TagID, assignment.DimensionID)
		}
		if assignment.Confidence < 0 || assignment.Confidence > 1 {
			return nil, fmt.Errorf("auto-tag confidence %.4f is outside [0,1]", assignment.Confidence)
		}
		seen[assignment.DimensionID] = struct{}{}
		rows = append(rows, &types.DocumentTag{
			TenantID:    payload.TenantID,
			DocumentID:  payload.KnowledgeID,
			TagID:       assignment.TagID,
			DimensionID: assignment.DimensionID,
			Confidence:  assignment.Confidence,
		})
	}
	return rows, nil
}
