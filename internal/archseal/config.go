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
	Version int                 `json:"version"`
	Roots   []string            `json:"roots"`
	Layers  map[string][]string `json:"layers"`
	Rules   []Rule              `json:"rules"`
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

	for _, rule := range cfg.Rules {
		if _, ok := cfg.Layers[rule.From]; !ok {
			return fmt.Errorf("rule references unknown layer %q", rule.From)
		}
		for _, denied := range rule.Deny {
			if _, ok := cfg.Layers[denied]; !ok {
				return fmt.Errorf("rule %q denies unknown layer %q", rule.From, denied)
			}
			if denied == rule.From {
				return fmt.Errorf("layer %q cannot deny itself", rule.From)
			}
		}
	}
	return nil
}
