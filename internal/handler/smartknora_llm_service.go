package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/models/utils/ollama"
	"github.com/Tencent/WeKnora/internal/types"
)

var ErrSmartKnoraLLMNotConfigured = errors.New("smartknora llm model is not configured")

type SmartKnoraLLMService struct {
	db            *gorm.DB
	ollamaService *ollama.OllamaService
}

type SmartKnoraLLMResult struct {
	Content          string
	ModelID          string
	ModelName        string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	FinishReason     string
}

func NewSmartKnoraLLMService(db *gorm.DB) *SmartKnoraLLMService {
	ollamaService, _ := ollama.GetOllamaService()
	return &SmartKnoraLLMService{db: db, ollamaService: ollamaService}
}

func (s *SmartKnoraLLMService) Generate(ctx context.Context, tenantID uint64, systemPrompt string, userPrompt string, maxTokens int) (*SmartKnoraLLMResult, error) {
	model, err := s.findDefaultChatModel(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(model.Parameters.APIKey) == "" && model.Source != types.ModelSourceLocal {
		return nil, fmt.Errorf("%w: model %s missing api_key", ErrSmartKnoraLLMNotConfigured, model.Name)
	}

	chatModel, err := chat.NewChat(chat.ConfigFromModel(model, model.Parameters.AppID, model.Parameters.AppSecret), s.ollamaService)
	if err != nil {
		return nil, err
	}
	if maxTokens <= 0 {
		maxTokens = 1200
	}
	messages := []chat.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	resp, err := chatModel.Chat(ctx, messages, &chat.ChatOptions{Temperature: 0.2, MaxTokens: maxTokens})
	if err != nil {
		return nil, err
	}
	return &SmartKnoraLLMResult{
		Content:          strings.TrimSpace(resp.Content),
		ModelID:          model.ID,
		ModelName:        model.Name,
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
		FinishReason:     resp.FinishReason,
	}, nil
}

func (s *SmartKnoraLLMService) findDefaultChatModel(ctx context.Context, tenantID uint64) (*types.Model, error) {
	var model types.Model
	query := s.db.WithContext(ctx).Where("(tenant_id = ? OR is_builtin = true) AND deleted_at IS NULL AND status = ?", tenantID, types.ModelStatusActive)
	query = query.Where("type IN ?", []types.ModelType{types.ModelTypeKnowledgeQA, types.ModelTypeVLLM, types.ModelType("llm")})
	if err := query.Where("is_default = true").Order("tenant_id DESC, updated_at DESC").First(&model).Error; err == nil {
		return normalizeSmartKnoraModel(&model), nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if err := query.Order("tenant_id DESC, updated_at DESC").First(&model).Error; err == nil {
		return normalizeSmartKnoraModel(&model), nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return nil, ErrSmartKnoraLLMNotConfigured
}

func normalizeSmartKnoraModel(model *types.Model) *types.Model {
	if model == nil {
		return nil
	}
	if model.Type == "llm" || strings.EqualFold(string(model.Type), "llm") {
		model.Type = types.ModelTypeKnowledgeQA
	}
	if model.Source == types.ModelSourceDeepseek {
		if model.Parameters.Provider == "" {
			model.Parameters.Provider = "deepseek"
		}
		if model.Parameters.BaseURL == "" {
			model.Parameters.BaseURL = "https://api.deepseek.com/v1"
		}
		model.Source = types.ModelSourceRemote
	}
	if model.Source != types.ModelSourceLocal {
		model.Source = types.ModelSourceRemote
	}
	if model.Parameters.Provider == "" && model.Source == types.ModelSourceOpenAI {
		model.Parameters.Provider = "openai"
	}
	if model.Parameters.Provider == "" && model.Parameters.BaseURL != "" {
		model.Parameters.Provider = "generic"
	}
	return model
}

func (s *SmartKnoraLLMService) recordUsage(tenantID uint64, userID string, modelID string, apiPath string, promptTokens, completionTokens, totalTokens int) {
	if s == nil || s.db == nil || totalTokens <= 0 {
		return
	}
	uid := userID
	usage := types.SmartKnoraTokenUsage{
		UserID:           &uid,
		TenantID:         tenantID,
		ModelID:          modelID,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		APIPath:          apiPath,
		CreatedAt:        time.Now(),
	}
	_ = s.db.Create(&usage).Error
}
