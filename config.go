package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ── Config structures ────────────────────────────────────────────────────

type LanguageConfig struct {
	Name       string            `yaml:"name"`
	Label      string            `yaml:"label"`
	Detect     []string          `yaml:"detect"`
	Frameworks []FrameworkConfig `yaml:"frameworks"`
}

type FrameworkConfig struct {
	Name       string     `yaml:"name"`
	Label      string     `yaml:"label"`
	Create     []string   `yaml:"create"`
	PostCreate [][]string `yaml:"post_create"`
}

type Config struct {
	Languages []LanguageConfig `yaml:"languages"`
}

// ── Default config embedded ──────────────────────────────────────────────

//go:embed default-config.yaml
var defaultConfigData []byte

// ── Load ─────────────────────────────────────────────────────────────────

func LoadConfig() (*Config, error) {
	var cfg Config

	// Unmarshal defaults
	if err := yaml.Unmarshal(defaultConfigData, &cfg); err != nil {
		return nil, fmt.Errorf("default config: %w", err)
	}

	// Merge user config from ~/.hf/config.yaml
	userPath := filepath.Join(hfConfigDir(), "config.yaml")
	data, err := os.ReadFile(userPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return nil, fmt.Errorf("reading %s: %w", userPath, err)
	}

	var userCfg Config
	if err := yaml.Unmarshal(data, &userCfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", userPath, err)
	}

	// Merge: user languages override defaults by name
	for _, ul := range userCfg.Languages {
		replaced := false
		for i, dl := range cfg.Languages {
			if dl.Name == ul.Name {
				cfg.Languages[i] = ul
				replaced = true
				break
			}
		}
		if !replaced {
			cfg.Languages = append(cfg.Languages, ul)
		}
	}

	return &cfg, nil
}

// ── Lookups ──────────────────────────────────────────────────────────────

func (c *Config) FindLanguage(name string) *LanguageConfig {
	for i := range c.Languages {
		if c.Languages[i].Name == name {
			return &c.Languages[i]
		}
	}
	return nil
}

func (c *LanguageConfig) FindFramework(name string) *FrameworkConfig {
	for i := range c.Frameworks {
		if c.Frameworks[i].Name == name {
			return &c.Frameworks[i]
		}
	}
	return nil
}

// LanguageLabel returns the display label for a language name.
// Falls back to the raw name if not found.
func (c *Config) LanguageLabel(name string) string {
	if name == "" || name == "unknown" {
		return "Unknown"
	}
	lang := c.FindLanguage(name)
	if lang != nil {
		return lang.Label
	}
	// Capitalise first letter as a fallback
	if len(name) > 0 {
		return strings.ToUpper(name[:1]) + name[1:]
	}
	return name
}

// FrameworkLabel returns the display label for a framework within a language.
func (c *Config) FrameworkLabel(langName, fwName string) string {
	if fwName == "" {
		return ""
	}
	lang := c.FindLanguage(langName)
	if lang != nil {
		fw := lang.FindFramework(fwName)
		if fw != nil {
			return fw.Label
		}
	}
	return fwName
}

// Describe returns a human-readable summary like "Python" or "JS/TS — Vite + React".
func (c *Config) Describe(langName, fwName string) string {
	label := c.LanguageLabel(langName)
	if fwName != "" {
		fwLabel := c.FrameworkLabel(langName, fwName)
		if fwLabel != "" {
			return label + " — " + fwLabel
		}
	}
	return label
}

// ── Type detection ───────────────────────────────────────────────────────

// DetectLanguage scans a directory for known files (pyproject.toml, package.json, etc.)
// and returns the matching language name, or "unknown".
func DetectLanguage(path string, config *Config) string {
	for _, lang := range config.Languages {
		for _, file := range lang.Detect {
			if _, err := os.Stat(filepath.Join(path, file)); err == nil {
				return lang.Name
			}
		}
	}
	return "unknown"
}
