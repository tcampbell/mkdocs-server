package build

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/tcampbell/mkdocs-server/internal/aggregate"
	"github.com/tcampbell/mkdocs-server/internal/assets"
	"github.com/tcampbell/mkdocs-server/internal/config"
)

// Build generates the static site from the config at configPath.
func Build(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// Resolve docs and site dirs relative to the config file location.
	docsDir := filepath.Join(cfg.ConfigDir, cfg.DocsDir)
	siteDir := filepath.Join(cfg.ConfigDir, cfg.SiteDir)

	// Multi-repo aggregation: when sources are defined, clone each source repo
	// and assemble a unified docs directory. The normal build pipeline then runs
	// against that directory using the merged nav.
	if len(cfg.Sources) > 0 {
		result, err := aggregate.Aggregate(cfg.Sources)
		if err != nil {
			return err
		}
		defer result.Cleanup()
		docsDir = result.DocsDir
		if len(cfg.Nav) == 0 {
			cfg.Nav = result.Nav
		}
	}

	if err := os.MkdirAll(siteDir, 0o755); err != nil {
		return fmt.Errorf("create site dir: %w", err)
	}

	if err := copyAssets(siteDir); err != nil {
		return fmt.Errorf("copy assets: %w", err)
	}

	var searchDocs []searchDoc

	err = filepath.WalkDir(docsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(docsDir, path)

		// Copy non-markdown files (images, etc.) verbatim; skip hidden config files.
		if !strings.HasSuffix(rel, ".md") {
			if strings.HasPrefix(filepath.Base(rel), ".") {
				return nil
			}
			dst := filepath.Join(siteDir, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			return copyFile(path, dst)
		}

		doc, err := processPage(cfg, docsDir, siteDir, rel)
		if err != nil {
			return fmt.Errorf("process %s: %w", rel, err)
		}
		searchDocs = append(searchDocs, doc)
		return nil
	})
	if err != nil {
		return err
	}

	if err := writeSearchIndex(siteDir, searchDocs); err != nil {
		return fmt.Errorf("write search index: %w", err)
	}

	fmt.Printf("built %d pages → %s\n", len(searchDocs), siteDir)
	return nil
}

// processPage renders one .md file and writes the HTML output.
// Returns a searchDoc for the search index.
func processPage(cfg *config.Config, docsDir, siteDir, relMDPath string) (searchDoc, error) {
	src, err := os.ReadFile(filepath.Join(docsDir, relMDPath))
	if err != nil {
		return searchDoc{}, err
	}

	meta, body := stripFrontmatter(string(src))
	title := extractTitle(body, meta)
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(relMDPath), ".md")
	}

	contentStr := renderMarkdown(body)

	// Derive output path: foo/bar.md → foo/bar.html
	outputRel := strings.TrimSuffix(filepath.ToSlash(relMDPath), ".md") + ".html"
	outputPath := filepath.Join(siteDir, filepath.FromSlash(outputRel))

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return searchDoc{}, err
	}

	currentURL := "/" + outputRel
	nav := RenderNav(cfg.Nav, currentURL)
	base := computeBase(outputRel)

	data := pageData{
		SiteName:   cfg.SiteName,
		PageTitle:  title,
		Content:    template.HTML(contentStr), //nolint:gosec — content rendered from trusted local files
		Nav:        nav,
		ExtraCSS:   cfg.ExtraCSS,
		ConfigJSON: buildConfigJSON(base, cfg.Theme.Features),
	}

	pageHTML, err := renderPage(data)
	if err != nil {
		return searchDoc{}, err
	}

	if err := os.WriteFile(outputPath, pageHTML, 0o644); err != nil {
		return searchDoc{}, err
	}

	doc := searchDoc{
		Location: currentURL,
		Title:    title,
		Text:     plainText(contentStr),
	}
	return doc, nil
}

// copyAssets copies the embedded Material assets into site/_assets/.
func copyAssets(siteDir string) error {
	return fs.WalkDir(assets.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		dst := filepath.Join(siteDir, "_assets", filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		f, err := assets.FS.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		out, err := os.Create(dst)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, f)
		return err
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
