package archseal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckDetectsForbiddenBoundary(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "src/domain/order.ts"), `import { db } from "../infrastructure/db";\nexport const order = db;\n`)
	mustWrite(t, filepath.Join(dir, "src/infrastructure/db.ts"), `export const db = {};\n`)

	cfg := Config{
		Version: 1,
		Roots:   []string{"src"},
		Layers: map[string][]string{
			"domain":         {"src/domain"},
			"infrastructure": {"src/infrastructure"},
		},
		Rules: []Rule{{From: "domain", Deny: []string{"infrastructure"}}},
	}
	configPath := filepath.Join(dir, ".archseal.json")
	writeConfig(t, configPath, cfg)

	report, err := Check(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(report.Violations); got != 1 {
		t.Fatalf("expected 1 violation, got %d: %#v", got, report.Violations)
	}
	v := report.Violations[0]
	if v.FromLayer != "domain" || v.ToLayer != "infrastructure" {
		t.Fatalf("unexpected edge: %#v", v)
	}
}

func TestCheckAllowsInwardDependency(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "src/application/usecase.ts"), `import { entity } from "../domain/entity";\nexport { entity };\n`)
	mustWrite(t, filepath.Join(dir, "src/domain/entity.ts"), `export const entity = 1;\n`)

	cfg := Config{
		Version: 1,
		Roots:   []string{"src"},
		Layers: map[string][]string{
			"domain":      {"src/domain"},
			"application": {"src/application"},
		},
		Rules: []Rule{{From: "domain", Deny: []string{"application"}}},
	}
	configPath := filepath.Join(dir, ".archseal.json")
	writeConfig(t, configPath, cfg)

	report, err := Check(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Violations) != 0 {
		t.Fatalf("expected clean report, got %#v", report.Violations)
	}
}

func TestConfigRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".archseal.json")
	mustWrite(t, path, `{"version":1,"roots":["src"],"layers":{"domain":["src/domain"]},"rules":[],"typo":true}`)
	_, _, err := loadConfig(path)
	if err == nil {
		t.Fatal("expected config error")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeConfig(t *testing.T, path string, cfg Config) {
	t.Helper()
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, path, string(raw))
}
