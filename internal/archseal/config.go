package archseal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Config struct {
	Version      int                 `json:"version"`
	Roots        []string            `json:"roots"`
	Layers       map[string][]string `json:"layers"`
	Rules        []Rule              `json:"rules"`
	Aliases      map[string]string   `json:"aliases,omitempty"`
	ForbidCycles bool                `json:"forbid_cycles,omitempty"`
}

type Rule struct {
	From string   `json:"from"`
	Deny []string `json:"deny"`
}

func DefaultConfig() Config {
	return Config{
		Version: 1,
		Roots:   []string{"src"},
		Layers: map[string][]string{
			"domain":         {"src/domain"},
			"application":    {"src/application"},
			"infrastructure": {"src/infrastructure"},
			"interfaces":     {"src/interfaces"},
		},
		Rules: []Rule{
			{From: "domain", Deny: []string{"application", "infrastructure", "interfaces"}},
			{From: "application", Deny: []string{"infrastructure", "interfaces"}},
			{From: "infrastructure", Deny: []string{"interfaces"}},
		},
		Aliases: map[string]string{
			"@domain/*": "src/domain/*",
		},
		ForbidCycles: true,
	}
}

func loadConfig(path string) (Config, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, "", fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, "", fmt.Errorf("parse config: %w", err)
	}
	if err := validateConfig(cfg); err != nil {
		return Config{}, "", err
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return Config{}, "", err
	}
	return cfg, filepath.Dir(abs), nil
}

func validateConfig(cfg Config) error {
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported config version %d", cfg.Version)
	}
	if len(cfg.Roots) == 0 {
		return errors.New("config roots cannot be empty")
	}
	if len(cfg.Layers) == 0 {
		return errors.New("config layers cannot be empty")
	}

	layerNames := make([]string, 0, len(cfg.Layers))
	for name, paths := range cfg.Layers {
		if strings.TrimSpace(name) == "" {
			return errors.New("layer name cannot be empty")
		}
		if len(paths) == 0 {
			return fmt.Errorf("layer %q has no paths", name)
		}
		layerNames = append(layerNames, name)
	}
	sort.Strings(layerNames)

	type ownedPath struct {
		layer string
		path  string
	}
	var owned []ownedPath
	for _, layer := range layerNames {
		for _, path := range cfg.Layers[layer] {
			p := cleanPolicyPath(path)
			if p == "" || p == "." {
				return fmt.Errorf("layer %q contains an invalid path %q", layer, path)
			}
			owned = append(owned, ownedPath{layer: layer, path: p})
		}
	}
	for i := 0; i < len(owned); i++ {
		for j := i + 1; j < len(owned); j++ {
			if owned[i].layer == owned[j].layer {
				continue
			}
			if pathsOverlap(owned[i].path, owned[j].path) {
				return fmt.Errorf(
					"layer paths overlap: %q (%s) and %q (%s)",
					owned[i].path, owned[i].layer, owned[j].path, owned[j].layer,
				)
			}
		}
	}

	for _, rule := range cfg.Rules {
		if _, ok := cfg.Layers[rule.From]; !ok {
			return fmt.Errorf("rule references unknown layer %q", rule.From)
		}
		seen := map[string]bool{}
		for _, denied := range rule.Deny {
			if _, ok := cfg.Layers[denied]; !ok {
				return fmt.Errorf("rule %q denies unknown layer %q", rule.From, denied)
			}
			if denied == rule.From {
				return fmt.Errorf("layer %q cannot deny itself", rule.From)
			}
			if seen[denied] {
				return fmt.Errorf("rule %q denies layer %q more than once", rule.From, denied)
			}
			seen[denied] = true
		}
	}

	for pattern, target := range cfg.Aliases {
		if err := validateAlias(pattern, target); err != nil {
			return err
		}
	}
	return nil
}

func validateAlias(pattern, target string) error {
	if pattern == "" || target == "" {
		return errors.New("alias pattern and target cannot be empty")
	}
	pStars := strings.Count(pattern, "*")
	tStars := strings.Count(target, "*")
	if pStars != tStars || pStars > 1 {
		return fmt.Errorf("alias %q -> %q must use zero or one matching wildcard", pattern, target)
	}
	if pStars == 1 && (!strings.HasSuffix(pattern, "*") || !strings.HasSuffix(target, "*")) {
		return fmt.Errorf("alias %q -> %q wildcard must be the final character", pattern, target)
	}
	if strings.HasPrefix(target, "/") || filepath.IsAbs(target) || strings.HasPrefix(cleanPolicyPath(target), "../") {
		return fmt.Errorf("alias %q target must stay inside the repository", pattern)
	}
	return nil
}

func cleanPolicyPath(path string) string {
	return strings.TrimPrefix(filepath.ToSlash(filepath.Clean(path)), "./")
}

func pathsOverlap(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}
