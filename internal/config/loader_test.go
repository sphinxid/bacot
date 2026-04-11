package config

import (
	"os"
	"testing"
	"time"
)

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "bacot-test-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestLoad_ValidScript(t *testing.T) {
	yaml := `
name: "Test"
target: "https://example.com"
stages:
  - duration: 10s
    vus: 5
scenarios:
  - name: "GET home"
    method: GET
    path: /
    weight: 100
thresholds:
  http_req_duration_p95: "< 500ms"
`
	path := writeTempYAML(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Name != "Test" {
		t.Errorf("Name: want Test, got %s", cfg.Name)
	}
	if cfg.Target != "https://example.com" {
		t.Errorf("Target: want https://example.com, got %s", cfg.Target)
	}
	if len(cfg.Stages) != 1 {
		t.Fatalf("want 1 stage, got %d", len(cfg.Stages))
	}
	if cfg.Stages[0].VUs != 5 {
		t.Errorf("VUs: want 5, got %d", cfg.Stages[0].VUs)
	}
	if cfg.Stages[0].Duration.Duration != 10*time.Second {
		t.Errorf("Duration: want 10s, got %s", cfg.Stages[0].Duration.Duration)
	}
	if len(cfg.Scenarios) != 1 {
		t.Fatalf("want 1 scenario, got %d", len(cfg.Scenarios))
	}
	if cfg.Scenarios[0].Method != "GET" {
		t.Errorf("Method: want GET, got %s", cfg.Scenarios[0].Method)
	}
}

func TestLoad_DefaultsApplied(t *testing.T) {
	yaml := `
target: "https://example.com"
stages:
  - duration: 5s
    vus: 1
scenarios:
  - name: "test"
    path: /
`
	path := writeTempYAML(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("default timeout: want 30s, got %s", cfg.Timeout)
	}
	if cfg.ConnectTimeout != 10*time.Second {
		t.Errorf("default connect timeout: want 10s, got %s", cfg.ConnectTimeout)
	}
	if cfg.MaxRedirects != 10 {
		t.Errorf("default max redirects: want 10, got %d", cfg.MaxRedirects)
	}
	if cfg.Scenarios[0].Method != "GET" {
		t.Errorf("default method: want GET, got %s", cfg.Scenarios[0].Method)
	}
}

func TestLoad_InvalidStage(t *testing.T) {
	yaml := `
target: "https://example.com"
stages:
  - duration: 0s
    vus: 5
`
	path := writeTempYAML(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for zero-duration stage")
	}
}

func TestLoad_InvalidVUs(t *testing.T) {
	yaml := `
target: "https://example.com"
stages:
  - duration: 10s
    vus: 0
`
	path := writeTempYAML(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for zero VUs")
	}
}

func TestLoadInline_Defaults(t *testing.T) {
	cfg, err := LoadInline("https://example.com", 0, 0)
	if err != nil {
		t.Fatalf("LoadInline failed: %v", err)
	}
	if cfg.InlineVUs != 1 {
		t.Errorf("default VUs: want 1, got %d", cfg.InlineVUs)
	}
	if cfg.InlineDuration != 30*time.Second {
		t.Errorf("default duration: want 30s, got %s", cfg.InlineDuration)
	}
}

func TestLoadInline_MissingURL(t *testing.T) {
	_, err := LoadInline("", 5, 30*time.Second)
	if err == nil {
		t.Error("expected error for missing URL")
	}
}

func TestTotalDuration(t *testing.T) {
	cfg := &Config{
		Stages: []Stage{
			{Duration: Duration{Duration: 10 * time.Second}},
			{Duration: Duration{Duration: 30 * time.Second}},
			{Duration: Duration{Duration: 10 * time.Second}},
		},
	}
	if got := cfg.TotalDuration(); got != 50*time.Second {
		t.Errorf("TotalDuration: want 50s, got %s", got)
	}
}
