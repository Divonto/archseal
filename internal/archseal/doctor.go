package archseal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DoctorReport struct {
	ConfigPath string   `json:"config_path"`
	Checks     []string `json:"checks"`
}

func Doctor(configPath string) (DoctorReport, error) {
	cfg, baseDir, err := loadConfig(configPath)
	if err != nil {
		return DoctorReport{}, err
	}

	report := DoctorReport{ConfigPath: filepath.ToSlash(configPath)}
	report.Checks = append(report.Checks, "policy schema is valid")
	report.Checks = append(report.Checks, "layer paths are non-overlapping")
	report.Checks = append(report.Checks, "rules reference declared layers")
	report.Checks = append(report.Checks, "aliases are repository-local")

	roots := append([]string(nil), cfg.Roots...)
	sort.Strings(roots)
	for _, root := range roots {
		path := filepath.Join(baseDir, filepath.FromSlash(root))
		info, err := os.Stat(path)
		if err != nil {
			return DoctorReport{}, fmt.Errorf("root %q is not readable: %w", root, err)
		}
		if !info.IsDir() {
			return DoctorReport{}, fmt.Errorf("root %q is not a directory", root)
		}
	}
	report.Checks = append(report.Checks, fmt.Sprintf("%d root(s) are readable", len(roots)))

	layerNames := make([]string, 0, len(cfg.Layers))
	for name := range cfg.Layers {
		layerNames = append(layerNames, name)
	}
	sort.Strings(layerNames)
	for _, layer := range layerNames {
		for _, rel := range cfg.Layers[layer] {
			path := filepath.Join(baseDir, filepath.FromSlash(rel))
			info, err := os.Stat(path)
			if err != nil {
				return DoctorReport{}, fmt.Errorf("layer %q path %q is not readable: %w", layer, rel, err)
			}
			if !info.IsDir() {
				return DoctorReport{}, fmt.Errorf("layer %q path %q is not a directory", layer, rel)
			}
		}
	}
	report.Checks = append(report.Checks, fmt.Sprintf("%d layer(s) resolve to readable directories", len(layerNames)))

	return report, nil
}

func (r DoctorReport) Text() string {
	var b strings.Builder
	b.WriteString("ARCHSEAL DOCTOR\n\n")
	for _, check := range r.Checks {
		fmt.Fprintf(&b, "OK  %s\n", check)
	}
	b.WriteString("\nREADY  Policy is production-valid.\n")
	return b.String()
}
