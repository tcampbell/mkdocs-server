package build

import (
	"fmt"
	"html/template"
	"path/filepath"
	"strings"

	"github.com/tcampbell/mkdocs-server/internal/config"
)

// navPathToURL converts a docs-relative .md path to a site-relative HTML URL.
// "index.md" → "index.html", "framework/overview.md" → "framework/overview.html"
func navPathToURL(mdPath string) string {
	p := filepath.ToSlash(mdPath)
	if strings.HasSuffix(p, ".md") {
		return p[:len(p)-3] + ".html"
	}
	return p
}

var navCounter int

// RenderNav returns the <ul> block for the primary navigation sidebar.
func RenderNav(items []config.NavItem, currentURL string) template.HTML {
	navCounter = 0
	return template.HTML(renderNavItems(items, currentURL, 0))
}

func renderNavItems(items []config.NavItem, currentURL string, level int) string {
	var sb strings.Builder
	sb.WriteString(`<ul class="md-nav__list">`)
	for _, item := range items {
		sb.WriteString(renderNavItem(item, currentURL, level))
	}
	sb.WriteString(`</ul>`)
	return sb.String()
}

func renderNavItem(item config.NavItem, currentURL string, level int) string {
	if len(item.Children) == 0 {
		// Leaf
		url := navPathToURL(item.Path)
		active := ""
		if url == currentURL || "/"+url == currentURL {
			active = " md-nav__link--active"
		}
		return fmt.Sprintf(
			`<li class="md-nav__item"><a href="%s" class="md-nav__link%s">%s</a></li>`,
			template.HTMLEscapeString("/"+url),
			active,
			template.HTMLEscapeString(item.Title),
		)
	}

	// Section — check if any descendant is the active page
	expanded := sectionContainsActive(item.Children, currentURL)
	navCounter++
	id := fmt.Sprintf("__nav_%d", navCounter)

	checked := ""
	if expanded {
		checked = ` checked`
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<li class="md-nav__item md-nav__item--section">`, ))
	sb.WriteString(fmt.Sprintf(`<input class="md-nav__toggle md-toggle" type="checkbox" id="%s"%s>`, id, checked))
	sb.WriteString(fmt.Sprintf(
		`<label class="md-nav__link" for="%s"><span class="md-ellipsis">%s</span></label>`,
		id,
		template.HTMLEscapeString(item.Title),
	))
	sb.WriteString(fmt.Sprintf(`<nav class="md-nav" aria-label="%s" data-md-level="%d">`,
		template.HTMLEscapeString(item.Title), level+1))
	sb.WriteString(fmt.Sprintf(`<label class="md-nav__title" for="%s">%s</label>`,
		id, template.HTMLEscapeString(item.Title)))
	sb.WriteString(renderNavItems(item.Children, currentURL, level+1))
	sb.WriteString(`</nav></li>`)
	return sb.String()
}

func sectionContainsActive(items []config.NavItem, currentURL string) bool {
	for _, item := range items {
		if len(item.Children) == 0 {
			url := navPathToURL(item.Path)
			if url == currentURL || "/"+url == currentURL {
				return true
			}
		} else if sectionContainsActive(item.Children, currentURL) {
			return true
		}
	}
	return false
}
