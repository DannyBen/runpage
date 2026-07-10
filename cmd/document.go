package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type documentConfig struct {
	Required     []string `yaml:"required"`
	Dependencies []string `yaml:"dependencies"`
	Workdir      string   `yaml:"workdir"`
}

type loadedDocument struct {
	markdown []byte
	config   documentConfig
}

type frontMatter struct {
	Runpage documentConfig `yaml:"runpage"`
}

func loadDocument(path string, values documentValues) (loadedDocument, error) {
	markdown, err := os.ReadFile(path)
	if err != nil {
		return loadedDocument{}, fmt.Errorf("read document %s: %w", path, err)
	}
	config, markdown, err := parseFrontMatter(markdown)
	if err != nil {
		return loadedDocument{}, fmt.Errorf("parse front matter in %s: %w", path, err)
	}
	if err := validateDocumentConfig(config); err != nil {
		return loadedDocument{}, fmt.Errorf("invalid front matter in %s: %w", path, err)
	}
	return loadedDocument{markdown: injectDocumentValues(markdown, values), config: config}, nil
}

func parseFrontMatter(markdown []byte) (documentConfig, []byte, error) {
	lines := bytes.SplitAfter(markdown, []byte("\n"))
	if len(lines) == 0 || strings.TrimSpace(string(lines[0])) != "---" {
		return documentConfig{}, markdown, nil
	}
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(string(lines[index])) != "---" {
			continue
		}
		var matter frontMatter
		if err := yaml.Unmarshal(bytes.Join(lines[1:index], nil), &matter); err != nil {
			return documentConfig{}, nil, err
		}
		return matter.Runpage, bytes.Join(lines[index+1:], nil), nil
	}
	return documentConfig{}, nil, fmt.Errorf("front matter is missing its closing ---")
}

func validateDocumentConfig(config documentConfig) error {
	seen := make(map[string]bool)
	for _, key := range config.Required {
		if !validValueKey(key) {
			return fmt.Errorf("invalid required value key: %s", key)
		}
		if seen[key] {
			return fmt.Errorf("duplicate required value: %s", key)
		}
		seen[key] = true
	}
	seen = make(map[string]bool)
	for _, dependency := range config.Dependencies {
		if strings.TrimSpace(dependency) == "" {
			return fmt.Errorf("dependency names cannot be empty")
		}
		if seen[dependency] {
			return fmt.Errorf("duplicate dependency: %s", dependency)
		}
		seen[dependency] = true
	}
	return nil
}

func requireDocumentValues(config documentConfig, values documentValues) error {
	missing := make([]string, 0)
	for _, key := range config.Required {
		if value, ok := values[key]; !ok || value == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required document values: %s (use --read to open without them)", strings.Join(missing, ", "))
	}
	return nil
}

func requireDependencies(config documentConfig) error {
	for _, dependency := range config.Dependencies {
		if _, err := exec.LookPath(dependency); err != nil {
			return fmt.Errorf("required dependency not found: %s", dependency)
		}
	}
	return nil
}

func resolveDocumentWorkdir(config documentConfig, override, documentPath string) (string, error) {
	setting := config.Workdir
	if override != "" {
		setting = override
	}
	if setting == "self" || strings.HasPrefix(setting, "self/") {
		absoluteDocument, err := filepath.Abs(documentPath)
		if err != nil {
			return "", fmt.Errorf("resolve document path %s: %w", documentPath, err)
		}
		relative := strings.TrimPrefix(setting, "self")
		relative = strings.TrimPrefix(relative, "/")
		return resolveWorkdir(filepath.Join(filepath.Dir(absoluteDocument), filepath.FromSlash(relative)))
	}
	return resolveWorkdir(setting)
}
