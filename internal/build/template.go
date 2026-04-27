package build

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
	"path/filepath"
	"strings"
)

//go:embed tmpl/page.html
var tmplFS embed.FS

var pageTmpl = template.Must(
	template.New("page.html").ParseFS(tmplFS, "tmpl/page.html"),
)

type pageData struct {
	SiteName   string
	PageTitle  string
	Content    template.HTML
	Nav        template.HTML
	ExtraCSS   []string
	ConfigJSON template.JS // raw JSON for <script id="__config">
}

type materialConfig struct {
	Base         string            `json:"base"`
	Features     []string          `json:"features"`
	Search       string            `json:"search"`
	Translations map[string]string `json:"translations"`
	Version      any               `json:"version"`
}

func renderPage(data pageData) ([]byte, error) {
	var buf bytes.Buffer
	if err := pageTmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// buildConfigJSON produces the JSON blob for <script id="__config">.
// base is the relative path from the current page to the site root (e.g. ".." or ".").
func buildConfigJSON(base string, features []string) template.JS {
	cfg := materialConfig{
		Base:         base,
		Features:     features,
		Search:       "/_assets/javascripts/workers/search.min.js",
		Translations: map[string]string{},
		Version:      nil,
	}
	if cfg.Features == nil {
		cfg.Features = []string{}
	}
	b, _ := json.Marshal(cfg)
	return template.JS(b)
}

// computeBase returns the relative path from an output HTML file to the site root.
// "index.html" → "."
// "framework/overview.html" → ".."
func computeBase(outputRelPath string) string {
	dir := filepath.ToSlash(filepath.Dir(outputRelPath))
	if dir == "." {
		return "."
	}
	parts := strings.Split(dir, "/")
	up := make([]string, len(parts))
	for i := range up {
		up[i] = ".."
	}
	return strings.Join(up, "/")
}
