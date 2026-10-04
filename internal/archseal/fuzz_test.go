package archseal

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzExtractImportsNeverPanics(f *testing.F) {
	f.Add(`import x from "./x"`)
	f.Add(`const x = require("../x")`)
	f.Add(`export { x } from "@core/x"`)
	f.Add("")

	f.Fuzz(func(t *testing.T, source string) {
		_ = extractImports(source)
	})
}

func FuzzAliasResolutionStaysInsideRepository(f *testing.F) {
	f.Add("@core/entity")
	f.Add("@core/../../escape")
	f.Add("")
	f.Add("./relative")

	f.Fuzz(func(t *testing.T, spec string) {
		dir := t.TempDir()
		source := filepath.Join(dir, "src", "domain", "source.ts")
		target := filepath.Join(dir, "src", "core", "entity.ts")

		if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(source, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, nil, 0o644); err != nil {
			t.Fatal(err)
		}

		resolved := resolveSpecifier(
			dir,
			source,
			spec,
			map[string]string{"@core/*": "src/core/*"},
		)
		if resolved == "" {
			return
		}

		rel, err := filepath.Rel(dir, resolved)
		if err != nil {
			t.Fatal(err)
		}
		if rel == ".." || (len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator)) {
			t.Fatalf("resolver escaped repository: %q -> %q", spec, resolved)
		}
	})
}
