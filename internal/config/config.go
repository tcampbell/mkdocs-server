package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	SiteName  string `yaml:"site_name"`
	SiteURL   string `yaml:"site_url"`
	DocsDir   string `yaml:"docs_dir"`
	SiteDir   string `yaml:"site_dir"`
	Theme     struct {
		Name     string   `yaml:"name"`
		Features []string `yaml:"features"`
	} `yaml:"theme"`
	ExtraCSS []string  `yaml:"extra_css"`
	Nav      []NavItem `yaml:"-"`
	Inherit  string    `yaml:"INHERIT"`

	// directory containing the config file — used to resolve relative paths
	ConfigDir string `yaml:"-"`
}

type NavItem struct {
	Title    string
	Path     string    // relative .md path (leaf only)
	Children []NavItem // non-empty for sections
}

// rawConfig mirrors Config but keeps nav as raw YAML for deferred parsing.
type rawConfig struct {
	SiteName string `yaml:"site_name"`
	SiteURL  string `yaml:"site_url"`
	DocsDir  string `yaml:"docs_dir"`
	SiteDir  string `yaml:"site_dir"`
	Theme    struct {
		Name     string   `yaml:"name"`
		Features []string `yaml:"features"`
	} `yaml:"theme"`
	ExtraCSS []string `yaml:"extra_css"`
	Nav      []any    `yaml:"nav"`
	Inherit  string   `yaml:"INHERIT"`
}

func Load(configPath string) (*Config, error) {
	abs, err := filepath.Abs(configPath)
	if err != nil {
		return nil, err
	}
	return load(abs, map[string]bool{})
}

func load(absPath string, seen map[string]bool) (*Config, error) {
	if seen[absPath] {
		return nil, fmt.Errorf("INHERIT cycle detected: %s", absPath)
	}
	seen[absPath] = true

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", absPath, err)
	}

	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", absPath, err)
	}

	cfg := &Config{
		SiteName:  raw.SiteName,
		SiteURL:   raw.SiteURL,
		DocsDir:   raw.DocsDir,
		SiteDir:   raw.SiteDir,
		ExtraCSS:  raw.ExtraCSS,
		Inherit:   raw.Inherit,
		ConfigDir: filepath.Dir(absPath),
	}
	cfg.Theme.Name = raw.Theme.Name
	cfg.Theme.Features = raw.Theme.Features
	cfg.Nav = parseNav(raw.Nav)

	if raw.Inherit != "" {
		basePath := filepath.Join(cfg.ConfigDir, raw.Inherit)
		absBase, err := filepath.Abs(basePath)
		if err != nil {
			return nil, err
		}
		base, err := load(absBase, seen)
		if err != nil {
			return nil, fmt.Errorf("load inherited config %s: %w", basePath, err)
		}
		cfg.mergeFrom(base)
	}

	if cfg.DocsDir == "" {
		cfg.DocsDir = "docs"
	}
	if cfg.SiteDir == "" {
		cfg.SiteDir = "site"
	}

	return cfg, nil
}

func (c *Config) mergeFrom(base *Config) {
	if c.SiteName == "" {
		c.SiteName = base.SiteName
	}
	if c.SiteURL == "" {
		c.SiteURL = base.SiteURL
	}
	if c.DocsDir == "" {
		c.DocsDir = base.DocsDir
	}
	if c.SiteDir == "" {
		c.SiteDir = base.SiteDir
	}
	if c.Theme.Name == "" {
		c.Theme.Name = base.Theme.Name
		c.Theme.Features = base.Theme.Features
	}
	if len(c.ExtraCSS) == 0 {
		c.ExtraCSS = base.ExtraCSS
	}
	if len(c.Nav) == 0 {
		c.Nav = base.Nav
	}
}

func parseNav(raw []any) []NavItem {
	items := make([]NavItem, 0, len(raw))
	for _, entry := range raw {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		for title, val := range m {
			switch v := val.(type) {
			case string:
				items = append(items, NavItem{Title: title, Path: v})
			case []any:
				items = append(items, NavItem{Title: title, Children: parseNav(v)})
			}
		}
	}
	return items
}
