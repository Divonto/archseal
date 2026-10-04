package archseal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

type Cycle struct {
	Files []string `json:"files"`
}

type Report struct {
	FilesScanned int         `json:"files_scanned"`
	EdgesChecked int         `json:"edges_checked"`
	Violations   []Violation `json:"violations"`
	Cycles       []Cycle     `json:"cycles"`
	Seal         string      `json:"seal"`
}

func (r Report) Failed() bool {
	return len(r.Violations) > 0 || len(r.Cycles) > 0
}

func (r Report) Text() string {
	var b strings.Builder
	b.WriteString("ARCHSEAL v1\n\n")
	fmt.Fprintf(&b, "Files scanned: %d\n", r.FilesScanned)
	fmt.Fprintf(&b, "Import edges:  %d\n", r.EdgesChecked)
	fmt.Fprintf(&b, "Violations:    %d\n", len(r.Violations))
	fmt.Fprintf(&b, "Cycles:        %d\n", len(r.Cycles))
	fmt.Fprintf(&b, "Seal:          %s\n", r.Seal)

	for _, v := range r.Violations {
		fmt.Fprintf(&b, "\nDENY  %s:%d\n", v.File, v.Line)
		fmt.Fprintf(&b, "      %s -> %s via %q\n", v.FromLayer, v.ToLayer, v.ImportPath)
	}
	for _, cycle := range r.Cycles {
		fmt.Fprintf(&b, "\nCYCLE %s\n", strings.Join(cycle.Files, " -> "))
	}

	if r.Failed() {
		b.WriteString("\nOPEN  Architecture contract failed.\n")
	} else {
		b.WriteString("\nSEALED  Architecture contract holds.\n")
	}
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
	graph := make(map[string][]string, len(files))

	for _, absPath := range files {
		rel, err := filepath.Rel(baseDir, absPath)
		if err != nil {
			return Report{}, err
		}
		rel = filepath.ToSlash(rel)
		graph[rel] = nil

		raw, err := os.ReadFile(absPath)
		if err != nil {
			return Report{}, err
		}
		content := string(raw)
		fromLayer := classify(rel, cfg.Layers)

		for _, imp := range extractImports(content) {
			resolved := resolveSpecifier(baseDir, absPath, imp.Specifier, cfg.Aliases)
			if resolved == "" {
				continue
			}
			targetRel, err := filepath.Rel(baseDir, resolved)
			if err != nil {
				continue
			}
			targetRel = filepath.ToSlash(targetRel)
			if strings.HasPrefix(targetRel, "../") {
				continue
			}

			report.EdgesChecked++
			graph[rel] = append(graph[rel], targetRel)

			toLayer := classify(targetRel, cfg.Layers)
			if fromLayer == "" || toLayer == "" || toLayer == fromLayer {
				continue
			}
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

	if cfg.ForbidCycles {
		report.Cycles = findCycles(graph)
	}
	report.Seal, err = computeSeal(cfg, baseDir, files)
	if err != nil {
		return Report{}, err
	}

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
	rel = cleanPolicyPath(rel)
	var names []string
	for name := range layers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		prefixes := append([]string(nil), layers[name]...)
		sort.Strings(prefixes)
		for _, prefix := range prefixes {
			p := strings.TrimSuffix(cleanPolicyPath(prefix), "/")
			if rel == p || strings.HasPrefix(rel, p+"/") {
				return name
			}
		}
	}
	return ""
}

func resolveSpecifier(baseDir, fromFile, spec string, aliases map[string]string) string {
	var base string
	if strings.HasPrefix(spec, ".") {
		base = filepath.Clean(filepath.Join(filepath.Dir(fromFile), filepath.FromSlash(spec)))
	} else {
		mapped, ok := applyAlias(spec, aliases)
		if !ok {
			return ""
		}
		base = filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(mapped)))
	}

	absBase, err := filepath.Abs(base)
	if err != nil {
		return ""
	}
	absRepo, err := filepath.Abs(baseDir)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(absRepo, absBase)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	return resolveCandidate(absBase)
}

func applyAlias(spec string, aliases map[string]string) (string, bool) {
	patterns := make([]string, 0, len(aliases))
	for pattern := range aliases {
		patterns = append(patterns, pattern)
	}
	sort.Slice(patterns, func(i, j int) bool {
		if len(patterns[i]) != len(patterns[j]) {
			return len(patterns[i]) > len(patterns[j])
		}
		return patterns[i] < patterns[j]
	})

	for _, pattern := range patterns {
		target := aliases[pattern]
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(spec, prefix) {
				suffix := strings.TrimPrefix(spec, prefix)
				return strings.TrimSuffix(target, "*") + suffix, true
			}
			continue
		}
		if spec == pattern {
			return target, true
		}
	}
	return "", false
}

func resolveCandidate(base string) string {
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
	return strings.Count(content[:offset], "\n") + 1
}

func findCycles(graph map[string][]string) []Cycle {
	nodes := make([]string, 0, len(graph))
	for node := range graph {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	for node := range graph {
		graph[node] = uniqueSorted(graph[node])
	}

	index := 0
	indices := map[string]int{}
	lowlink := map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	var cycles []Cycle

	var strongConnect func(string)
	strongConnect = func(v string) {
		indices[v] = index
		lowlink[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true

		for _, w := range graph[v] {
			if _, exists := graph[w]; !exists {
				continue
			}
			if _, seen := indices[w]; !seen {
				strongConnect(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
			} else if onStack[w] && indices[w] < lowlink[v] {
				lowlink[v] = indices[w]
			}
		}

		if lowlink[v] != indices[v] {
			return
		}

		var component []string
		for {
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[n] = false
			component = append(component, n)
			if n == v {
				break
			}
		}
		sort.Strings(component)

		if len(component) > 1 || hasSelfEdge(graph, component[0]) {
			cycles = append(cycles, Cycle{Files: component})
		}
	}

	for _, node := range nodes {
		if _, seen := indices[node]; !seen {
			strongConnect(node)
		}
	}

	sort.Slice(cycles, func(i, j int) bool {
		return strings.Join(cycles[i].Files, "\x00") < strings.Join(cycles[j].Files, "\x00")
	})
	return cycles
}

func hasSelfEdge(graph map[string][]string, node string) bool {
	for _, target := range graph[node] {
		if target == node {
			return true
		}
	}
	return false
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func computeSeal(cfg Config, baseDir string, files []string) (string, error) {
	h := sha256.New()
	canonical, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	h.Write([]byte("archseal:v1\x00"))
	h.Write(canonical)

	for _, file := range files {
		rel, err := filepath.Rel(baseDir, file)
		if err != nil {
			return "", err
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		h.Write([]byte{0})
		h.Write([]byte(filepath.ToSlash(rel)))
		h.Write([]byte{0})
		h.Write(raw)
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
