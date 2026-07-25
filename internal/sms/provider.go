package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	ProviderAliyun  = "aliyun"
	ProviderTencent = "tencent"
	ProviderHuawei  = "huawei"
	ProviderCustom  = "custom"
)

// Provider is the transport-independent contract used by verification and
// notification services to send an SMS message.
type Provider interface {
	Send(ctx context.Context, message Message) error
}

type Message struct {
	Phone          string
	TemplateParams map[string]string
}

// Config is populated from the sms_* rows in system_configs.
type Config struct {
	Provider        string
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	Region          string
	SignName        string
	TemplateID      string
	AppID           string
	Sender          string
	CustomHeaders   map[string]string
	Timeout         time.Duration
}

// ConfigFromValues converts system_configs key/value rows into provider config.
func ConfigFromValues(values map[string]string) (Config, error) {
	timeout := 10 * time.Second
	if raw := strings.TrimSpace(values["sms_timeout_seconds"]); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds <= 0 {
			return Config{}, fmt.Errorf("invalid sms_timeout_seconds %q", raw)
		}
		timeout = time.Duration(seconds) * time.Second
	}

	headers := map[string]string{}
	if raw := strings.TrimSpace(values["sms_custom_headers"]); raw != "" && raw != "{}" {
		if err := json.Unmarshal([]byte(raw), &headers); err != nil {
			return Config{}, fmt.Errorf("invalid sms_custom_headers: %w", err)
		}
	}

	return Config{
		Provider:        strings.ToLower(strings.TrimSpace(values["sms_provider"])),
		Endpoint:        strings.TrimSpace(values["sms_endpoint"]),
		AccessKeyID:     strings.TrimSpace(values["sms_access_key_id"]),
		AccessKeySecret: strings.TrimSpace(values["sms_access_key_secret"]),
		Region:          strings.TrimSpace(values["sms_region"]),
		SignName:        strings.TrimSpace(values["sms_sign_name"]),
		TemplateID:      strings.TrimSpace(values["sms_template_id"]),
		AppID:           strings.TrimSpace(values["sms_app_id"]),
		Sender:          strings.TrimSpace(values["sms_sender"]),
		CustomHeaders:   headers,
		Timeout:         timeout,
	}, nil
}

func NewProvider(config Config) (Provider, error) {
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}
	if config.Endpoint == "" {
		return nil, errors.New("sms endpoint is required")
	}

	provider := &httpProvider{config: config, client: &http.Client{Timeout: config.Timeout}}
	switch config.Provider {
	case ProviderAliyun:
		provider.buildPayload = buildAliyunPayload
	case ProviderTencent:
		provider.buildPayload = buildTencentPayload
	case ProviderHuawei:
		provider.buildPayload = buildHuaweiPayload
	case ProviderCustom:
		provider.buildPayload = buildCustomPayload
	default:
		return nil, fmt.Errorf("unsupported sms provider %q", config.Provider)
	}
	return provider, nil
}

type payloadBuilder func(Config, Message) (map[string]any, error)

type httpProvider struct {
	config       Config
	client       *http.Client
	buildPayload payloadBuilder
}

func (p *httpProvider) Send(ctx context.Context, message Message) error {
	if strings.TrimSpace(message.Phone) == "" {
		return errors.New("sms phone is required")
	}
	payload, err := p.buildPayload(p.config, message)
	if err != nil {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal sms request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.Endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create sms request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.config.AccessKeyID != "" {
		req.Header.Set("X-SMS-Access-Key-ID", p.config.AccessKeyID)
	}
	if p.config.AccessKeySecret != "" {
		req.Header.Set("X-SMS-Access-Key-Secret", p.config.AccessKeySecret)
	}
	for key, value := range p.config.CustomHeaders {
		req.Header.Set(key, value)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("send sms: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("send sms: provider returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func requireTemplate(config Config) error {
	if config.TemplateID == "" {
		return errors.New("sms template id is required")
	}
	return nil
}

func buildAliyunPayload(config Config, message Message) (map[string]any, error) {
	if err := requireTemplate(config); err != nil {
		return nil, err
	}
	return map[string]any{
		"PhoneNumbers":  message.Phone,
		"SignName":      config.SignName,
		"TemplateCode":  config.TemplateID,
		"TemplateParam": message.TemplateParams,
		"RegionId":      config.Region,
	}, nil
}

func buildTencentPayload(config Config, message Message) (map[string]any, error) {
	if err := requireTemplate(config); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(message.TemplateParams))
	for key := range message.TemplateParams {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	params := make([]string, 0, len(keys))
	for _, key := range keys {
		params = append(params, message.TemplateParams[key])
	}
	return map[string]any{
		"PhoneNumberSet":   []string{message.Phone},
		"SmsSdkAppId":      config.AppID,
		"SignName":         config.SignName,
		"TemplateId":       config.TemplateID,
		"TemplateParamSet": params,
		"Region":           config.Region,
	}, nil
}

func buildHuaweiPayload(config Config, message Message) (map[string]any, error) {
	if err := requireTemplate(config); err != nil {
		return nil, err
	}
	return map[string]any{
		"to":            message.Phone,
		"from":          config.Sender,
		"signature":     config.SignName,
		"templateId":    config.TemplateID,
		"templateParas": message.TemplateParams,
	}, nil
}

func buildCustomPayload(config Config, message Message) (map[string]any, error) {
	return map[string]any{
		"phone":           message.Phone,
		"sign_name":       config.SignName,
		"template_id":     config.TemplateID,
		"template_params": message.TemplateParams,
	}, nil
}
