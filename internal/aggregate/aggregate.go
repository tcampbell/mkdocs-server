// Package aggregate implements multi-repo content fetching for mkdocs-server build.
// It shallow-clones each source repo, copies its docs_dir content into a
// temporary unified directory, and merges per-source nav trees.
package aggregate

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/tcampbell/mkdocs-server/internal/config"
)

// Result holds the merged docs directory and nav produced by Aggregate.
// Call Cleanup() when the build is done to remove the temporary directory.
type Result struct {
	DocsDir string
	Nav     []config.NavItem
	tempDir string
}

func (r *Result) Cleanup() {
	if r.tempDir != "" {
		os.RemoveAll(r.tempDir)
	}
}

// Aggregate shallow-clones each source, copies its content into a shared
// temporary directory, and merges nav entries.
// Sources that fail to clone hard-fail the build.
func Aggregate(sources []config.Source) (*Result, error) {
	tempDir, err := os.MkdirTemp("", "mkdocs-server-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	docsDir := filepath.Join(tempDir, "docs")
	var mergedNav []config.NavItem

	for _, source := range sources {
		fmt.Printf("fetching %s (%s @ %s)\n", source.Name, source.Repo, source.Ref)

		repoDir := filepath.Join(tempDir, "repos", source.Name)
		if err := clone(source, repoDir); err != nil {
			os.RemoveAll(tempDir)
			return nil, fmt.Errorf("clone source %q: %w", source.Name, err)
		}

		srcDocsDir := filepath.Join(repoDir, filepath.FromSlash(source.DocsDir))
		dstDocsDir := filepath.Join(docsDir, source.Name)
		if err := copyDir(srcDocsDir, dstDocsDir); err != nil {
			os.RemoveAll(tempDir)
			return nil, fmt.Errorf("copy docs for source %q: %w", source.Name, err)
		}

		children := sourceNav(source, repoDir)
		mergedNav = append(mergedNav, config.NavItem{
			Title:    source.Name,
			Children: children,
		})
	}

	return &Result{
		DocsDir: docsDir,
		Nav:     mergedNav,
		tempDir: tempDir,
	}, nil
}

// sourceNav reads the source repo's mkdocs.yml (searched adjacent to docs_dir)
// and returns its nav items prefixed with the source name.
func sourceNav(source config.Source, repoDir string) []config.NavItem {
	docsAbs := filepath.Join(repoDir, filepath.FromSlash(source.DocsDir))
	// mkdocs.yml conventionally lives one level above docs_dir
	cfgPath := filepath.Join(filepath.Dir(docsAbs), "mkdocs.yml")

	if _, err := os.Stat(cfgPath); err == nil {
		if cfg, err := config.Load(cfgPath); err == nil && len(cfg.Nav) > 0 {
			return prefixNav(cfg.Nav, source.Name+"/")
		}
	}
	return nil
}

// prefixNav prepends prefix to every leaf path in a nav tree.
func prefixNav(items []config.NavItem, prefix string) []config.NavItem {
	result := make([]config.NavItem, len(items))
	for i, item := range items {
		result[i] = item
		if item.Path != "" {
			result[i].Path = prefix + item.Path
		}
		if len(item.Children) > 0 {
			result[i].Children = prefixNav(item.Children, prefix)
		}
	}
	return result
}

// clone shallow-clones a source repo at the given ref into destDir.
func clone(source config.Source, destDir string) error {
	if err := os.MkdirAll(filepath.Dir(destDir), 0o755); err != nil {
		return err
	}
	cmd := exec.Command("git", "clone",
		"--depth", "1",
		"--branch", source.Ref,
		"--", source.Repo, destDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}
		return copyFile(path, dstPath)
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
