package utils

import (
	"reflect"
	"testing"
)

func TestMaskSensitiveData(t *testing.T) {
	input := map[string]any{
		"email":    "user@example.com",
		"password": "Secret1!",
		"nested": map[string]any{
			"access-token": "abc",
			"safe":         "value",
		},
		"items": []any{map[string]any{"api_key": "key"}},
	}

	got := MaskSensitiveData(input).(map[string]any)
	if got["email"] != "user@example.com" || got["password"] != "******" {
		t.Fatalf("unexpected masked output: %#v", got)
	}
	nested := got["nested"].(map[string]any)
	if nested["access-token"] != "******" || nested["safe"] != "value" {
		t.Fatalf("unexpected nested output: %#v", nested)
	}
	if !reflect.DeepEqual(input["password"], "Secret1!") {
		t.Fatal("MaskSensitiveData mutated its input")
	}
}
