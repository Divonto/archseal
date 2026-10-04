package archseal

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var sourceExt = map[string]bool{
	".ts": true, ".tsx": true, ".js": true, ".jsx": true,
	".mjs": true, ".cjs": true,
}

var importPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)\b(?:import|export)\s+(?:[^"']*?\s+from\s+)?["']([^"']+)["']`),
	regexp.MustCompile(`(?m)\brequire\(\s*["']([^"']+)["']\s*\)`),
	regexp.MustCompile(`(?m)\bimport\(\s*["']([^"']+)["']\s*\)`),
}

type Violation struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	FromLayer  string `json:"from_layer"`
	ToLayer    string `json:"to_layer"`
	ImportPath string `json:"import"`
}

type Report struct {
	FilesScanned int         `json:"files_scanned"`
	EdgesChecked int         `json:"edges_checked"`
	Violations   []Violation `json:"violations"`
}

func (r Report) Text() string {
	var b strings.Builder
	b.WriteString("ARCHSEAL\n\n")
	fmt.Fprintf(&b, "Files scanned: %d\n", r.FilesScanned)
	fmt.Fprintf(&b, "Import edges:  %d\n", r.EdgesChecked)

	if len(r.Violations) == 0 {
		b.WriteString("Violations:    0\n\n")
		b.WriteString("SEALED  Architecture boundaries hold.\n")
		return b.String()
	}

	fmt.Fprintf(&b, "Violations:    %d\n\n", len(r.Violations))
	for _, v := range r.Violations {
		fmt.Fprintf(&b, "DENY  %s:%d\n", v.File, v.Line)
		fmt.Fprintf(&b, "      %s -> %s via %q\n", v.FromLayer, v.ToLayer, v.ImportPath)
	}
	b.WriteString("\nOPEN  Architecture boundary violation detected.\n")
	return b.String()
}

func Check(configPath string) (Report, error) {
	cfg, baseDir, err := loadConfig(configPath)
	if err != nil {
		return Report{}, err
	}

	denied := make(map[string]map[string]bool)
	for _, rule := range cfg.Rules {
		if denied[rule.From] == nil {
			denied[rule.From] = make(map[string]bool)
		}
		for _, to := range rule.Deny {
			denied[rule.From][to] = true
		}
	}

	files, err := collectFiles(baseDir, cfg.Roots)
	if err != nil {
		return Report{}, err
	}

	report := Report{FilesScanned: len(files)}
	for _, absPath := range files {
		rel, err := filepath.Rel(baseDir, absPath)
		if err != nil {
			return Report{}, err
		}
		rel = filepath.ToSlash(rel)
		fromLayer := classify(rel, cfg.Layers)
		if fromLayer == "" {
			continue
		}

		raw, err := os.ReadFile(absPath)
		if err != nil {
			return Report{}, err
		}
		content := string(raw)
		imports := extractImports(content)
		for _, imp := range imports {
			if !strings.HasPrefix(imp.Specifier, ".") {
				continue
			}
			resolved := resolveImport(absPath, imp.Specifier)
			if resolved == "" {
				continue
			}
			targetRel, err := filepath.Rel(baseDir, resolved)
			if err != nil {
				continue
			}
			targetRel = filepath.ToSlash(targetRel)
			toLayer := classify(targetRel, cfg.Layers)
			if toLayer == "" || toLayer == fromLayer {
				continue
			}
			report.EdgesChecked++
			if denied[fromLayer][toLayer] {
				report.Violations = append(report.Violations, Violation{
					File:       rel,
					Line:       lineAt(content, imp.Offset),
					FromLayer:  fromLayer,
					ToLayer:    toLayer,
					ImportPath: imp.Specifier,
				})
			}
		}
	}

	sort.Slice(report.Violations, func(i, j int) bool {
		a, b := report.Violations[i], report.Violations[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.FromLayer != b.FromLayer {
			return a.FromLayer < b.FromLayer
		}
		return a.ImportPath < b.ImportPath
	})

	return report, nil
}

type importRef struct {
	Specifier string
	Offset    int
}

func extractImports(content string) []importRef {
	var refs []importRef
	seen := map[string]bool{}
	for _, re := range importPatterns {
		matches := re.FindAllStringSubmatchIndex(content, -1)
		for _, m := range matches {
			if len(m) < 4 || m[2] < 0 || m[3] < 0 {
				continue
			}
			spec := content[m[2]:m[3]]
			key := fmt.Sprintf("%d:%s", m[2], spec)
			if seen[key] {
				continue
			}
			seen[key] = true
			refs = append(refs, importRef{Specifier: spec, Offset: m[2]})
		}
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Offset < refs[j].Offset })
	return refs
}

func collectFiles(baseDir string, roots []string) ([]string, error) {
	var files []string
	seen := map[string]bool{}
	for _, root := range roots {
		start := filepath.Join(baseDir, filepath.FromSlash(root))
		err := filepath.WalkDir(start, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				name := d.Name()
				if name == ".git" || name == "node_modules" || name == "dist" || name == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}
			if !sourceExt[strings.ToLower(filepath.Ext(path))] {
				return nil
			}
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			if !seen[abs] {
				seen[abs] = true
				files = append(files, abs)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("scan root %q: %w", root, err)
		}
	}
	sort.Strings(files)
	return files, nil
}

func classify(rel string, layers map[string][]string) string {
	rel = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(rel)), "./")
	var names []string
	for name := range layers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		prefixes := append([]string(nil), layers[name]...)
		sort.Strings(prefixes)
		for _, prefix := range prefixes {
			p := strings.TrimSuffix(strings.TrimPrefix(filepath.ToSlash(filepath.Clean(prefix)), "./"), "/")
			if rel == p || strings.HasPrefix(rel, p+"/") {
				return name
			}
		}
	}
	return ""
}

func resolveImport(fromFile, spec string) string {
	base := filepath.Clean(filepath.Join(filepath.Dir(fromFile), filepath.FromSlash(spec)))
	candidates := []string{base}
	for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"} {
		candidates = append(candidates, base+ext)
	}
	for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"} {
		candidates = append(candidates, filepath.Join(base, "index"+ext))
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			abs, _ := filepath.Abs(candidate)
			return abs
		}
	}
	return ""
}

func lineAt(content string, offset int) int {
	if offset <= 0 {
		return 1
	}
	scanner := bufio.NewScanner(strings.NewReader(content[:offset]))
	lines := 0
	for scanner.Scan() {
		lines++
	}
	if strings.HasSuffix(content[:offset], "\n") {
		return lines + 1
	}
	if lines == 0 {
		return 1
	}
	return lines
}
