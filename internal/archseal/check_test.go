package archseal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckDetectsForbiddenBoundary(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "src/domain/order.ts"), "import { db } from \"../infrastructure/db\";\nexport const order = db;\n")
	mustWrite(t, filepath.Join(dir, "src/infrastructure/db.ts"), "export const db = {};\n")

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
	if v.FromLayer != "domain" || v.ToLayer != "infrastructure" || v.Line != 1 {
		t.Fatalf("unexpected violation: %#v", v)
	}
}

func TestCheckAllowsInwardDependency(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "src/application/usecase.ts"), "import { entity } from \"../domain/entity\";\nexport { entity };\n")
	mustWrite(t, filepath.Join(dir, "src/domain/entity.ts"), "export const entity = 1;\n")

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
	if report.Failed() {
		t.Fatalf("expected clean report, got %#v", report)
	}
}

func TestAliasResolutionEnforcesBoundary(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "src/domain/order.ts"), "import { db } from \"@infra/db\";\n")
	mustWrite(t, filepath.Join(dir, "src/infrastructure/db.ts"), "export const db = {};\n")

	cfg := Config{
		Version: 1,
		Roots:   []string{"src"},
		Layers: map[string][]string{
			"domain":         {"src/domain"},
			"infrastructure": {"src/infrastructure"},
		},
		Rules:   []Rule{{From: "domain", Deny: []string{"infrastructure"}}},
		Aliases: map[string]string{"@infra/*": "src/infrastructure/*"},
	}
	path := filepath.Join(dir, ".archseal.json")
	writeConfig(t, path, cfg)

	report, err := Check(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Violations) != 1 || report.Violations[0].ImportPath != "@infra/db" {
		t.Fatalf("alias boundary was not enforced: %#v", report.Violations)
	}
}

func TestCycleDetection(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "src/core/a.ts"), "import \"./b\";\n")
	mustWrite(t, filepath.Join(dir, "src/core/b.ts"), "import \"./a\";\n")

	cfg := Config{
		Version:      1,
		Roots:        []string{"src"},
		Layers:       map[string][]string{"core": {"src/core"}},
		ForbidCycles: true,
	}
	path := filepath.Join(dir, ".archseal.json")
	writeConfig(t, path, cfg)

	report, err := Check(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Cycles) != 1 {
		t.Fatalf("expected one cycle, got %#v", report.Cycles)
	}
	if !report.Failed() {
		t.Fatal("cycle must fail the architecture contract")
	}
}

func TestSealIsDeterministicAndContentAddressed(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "src/domain/entity.ts")
	mustWrite(t, file, "export const entity = 1;\n")

	cfg := Config{
		Version: 1,
		Roots:   []string{"src"},
		Layers:  map[string][]string{"domain": {"src/domain"}},
	}
	path := filepath.Join(dir, ".archseal.json")
	writeConfig(t, path, cfg)

	first, err := Check(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Check(path)
	if err != nil {
		t.Fatal(err)
	}
	if first.Seal != second.Seal {
		t.Fatalf("seal is not deterministic: %s != %s", first.Seal, second.Seal)
	}
	if !strings.HasPrefix(first.Seal, "sha256:") {
		t.Fatalf("unexpected seal: %s", first.Seal)
	}

	mustWrite(t, file, "export const entity = 2;\n")
	changed, err := Check(path)
	if err != nil {
		t.Fatal(err)
	}
	if first.Seal == changed.Seal {
		t.Fatal("seal must change when analyzed source changes")
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

func TestConfigRejectsOverlappingLayers(t *testing.T) {
	cfg := Config{
		Version: 1,
		Roots:   []string{"src"},
		Layers: map[string][]string{
			"one": {"src"},
			"two": {"src/two"},
		},
	}
	if err := validateConfig(cfg); err == nil {
		t.Fatal("expected overlapping layer error")
	}
}

func TestDoctorAndSARIF(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "src/domain/entity.ts"), "export const entity = 1;\n")
	cfg := Config{
		Version: 1,
		Roots:   []string{"src"},
		Layers:  map[string][]string{"domain": {"src/domain"}},
	}
	path := filepath.Join(dir, ".archseal.json")
	writeConfig(t, path, cfg)

	doctor, err := Doctor(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(doctor.Checks) < 5 {
		t.Fatalf("doctor returned too few checks: %#v", doctor.Checks)
	}

	report, err := Check(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(report.SARIF())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"version":"2.1.0"`) {
		t.Fatalf("invalid SARIF payload: %s", raw)
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
