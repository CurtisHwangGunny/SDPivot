package sms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProviderPayloads(t *testing.T) {
	providers := []string{ProviderAliyun, ProviderTencent, ProviderHuawei, ProviderCustom}
	for _, name := range providers {
		t.Run(name, func(t *testing.T) {
			var payload map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("X-Test-Token"); got != "token" {
					t.Errorf("custom header = %q", got)
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Errorf("decode payload: %v", err)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			provider, err := NewProvider(Config{
				Provider:      name,
				Endpoint:      server.URL,
				TemplateID:    "template",
				CustomHeaders: map[string]string{"X-Test-Token": "token"},
			})
			if err != nil {
				t.Fatalf("NewProvider: %v", err)
			}
			if err := provider.Send(context.Background(), Message{Phone: "+8613800000000", TemplateParams: map[string]string{"code": "123456"}}); err != nil {
				t.Fatalf("Send: %v", err)
			}
			if len(payload) == 0 {
				t.Fatal("expected provider payload")
			}
		})
	}
}

func TestConfigFromValues(t *testing.T) {
	config, err := ConfigFromValues(map[string]string{
		"sms_provider":        "Tencent",
		"sms_timeout_seconds": "15",
		"sms_custom_headers":  `{"Authorization":"Bearer token"}`,
	})
	if err != nil {
		t.Fatalf("ConfigFromValues: %v", err)
	}
	if config.Provider != ProviderTencent || config.Timeout.Seconds() != 15 {
		t.Fatalf("unexpected config: %#v", config)
	}
	if config.CustomHeaders["Authorization"] != "Bearer token" {
		t.Fatalf("unexpected headers: %#v", config.CustomHeaders)
	}
}
