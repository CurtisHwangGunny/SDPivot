package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

func decodeProductYAML(t *testing.T, content string) *ProductConfig {
	t.Helper()
	v := viper.New()
	v.SetConfigType("yaml")
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	cfg := Config{Product: DefaultProductConfig()}
	if err := v.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "yaml"
	}); err != nil {
		t.Fatal(err)
	}
	return cfg.Product
}

func TestDefaultProductConfig(t *testing.T) {
	product := DefaultProductConfig()
	if product.OPMode || product.Brand != "sdpivot" || !product.EnableLegacyAlias {
		t.Fatalf("unexpected defaults: %+v", product)
	}
}

func TestProductYAMLDefaults(t *testing.T) {
	for _, tc := range []struct {
		name   string
		yaml   string
		opMode bool
		brand  string
		legacy bool
	}{
		{name: "missing", yaml: "server:\n  port: 8080\n", brand: "sdpivot", legacy: true},
		{name: "partial", yaml: "product:\n  op_mode: true\n  brand: custom\n", opMode: true, brand: "custom", legacy: true},
		{name: "explicit false", yaml: "product:\n  enable_legacy_alias: false\n", brand: "sdpivot", legacy: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			product := decodeProductYAML(t, tc.yaml)
			if product == nil || product.OPMode != tc.opMode || product.Brand != tc.brand || product.EnableLegacyAlias != tc.legacy {
				t.Fatalf("unexpected product: %+v", product)
			}
		})
	}
}

func TestApplyProductEnvOverrides(t *testing.T) {
	for _, tc := range []struct {
		name       string
		opMode     string
		legacy     string
		brand      string
		wantOP     bool
		wantLegacy bool
		wantBrand  string
		wantErr    bool
	}{
		{name: "true and brand", opMode: " true ", legacy: " true ", brand: " acme ", wantOP: true, wantLegacy: true, wantBrand: "acme"},
		{name: "false", opMode: "false", legacy: " false ", wantLegacy: false, wantBrand: "sdpivot"},
		{name: "invalid bool", legacy: "not-bool", wantLegacy: true, wantBrand: "sdpivot", wantErr: true},
		{name: "blank brand fallback", brand: "   ", wantLegacy: true, wantBrand: "sdpivot"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OP_MODE", tc.opMode)
			t.Setenv("SDP_ENABLE_LEGACY_ALIAS", tc.legacy)
			t.Setenv("BRAND", tc.brand)
			product := DefaultProductConfig()
			err := ApplyProductEnvOverrides(product)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if product.OPMode != tc.wantOP || product.EnableLegacyAlias != tc.wantLegacy || product.Brand != tc.wantBrand {
				t.Fatalf("unexpected product: %+v", product)
			}
		})
	}
}

func TestValidateConfigProduct(t *testing.T) {
	if err := ValidateConfig(&Config{}); err == nil {
		t.Fatal("expected nil product validation error")
	}
	if err := ValidateConfig(&Config{Product: &ProductConfig{}}); err == nil {
		t.Fatal("expected empty brand validation error")
	}
}
