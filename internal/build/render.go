package build

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Table,
		extension.Strikethrough,
		extension.TaskList,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		// raw HTML in source passes through (needed for preprocessed callouts/tabs)
		html.WithUnsafe(),
	),
)

// renderMarkdown converts markdown source to HTML after preprocessing.
func renderMarkdown(src string) string {
	src = preprocessCallouts(src)
	src = preprocessTabs(src)
	return rewriteMDLinks(renderRaw(src))
}

// mdLinkRe matches href="..." attributes whose value ends in .md and has no scheme.
var mdLinkRe = regexp.MustCompile(`href="([^"]*\.md)"`)

// rewriteMDLinks rewrites relative .md hrefs to .html in rendered HTML output,
// so in-content markdown links like [text](page.md) resolve to the built files.
func rewriteMDLinks(s string) string {
	return mdLinkRe.ReplaceAllStringFunc(s, func(match string) string {
		href := match[6 : len(match)-1] // strip href=" and trailing "
		if strings.Contains(href, "://") {
			return match // leave external URLs alone
		}
		return `href="` + href[:len(href)-3] + `.html"`
	})
}

// renderBodyMarkdown renders a markdown fragment without callout/tab preprocessing,
// used when rendering already-extracted content (callout body, tab content).
func renderBodyMarkdown(src string) string {
	return renderRaw(src)
}

func renderRaw(src string) string {
	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		return "<pre>" + src + "</pre>"
	}
	return buf.String()
}

var frontmatterRe = regexp.MustCompile(`(?s)^---\n(.+?)\n---\n?`)

// stripFrontmatter removes YAML frontmatter and returns the body.
func stripFrontmatter(src string) (map[string]string, string) {
	m := frontmatterRe.FindStringSubmatch(src)
	if m == nil {
		return nil, src
	}
	meta := make(map[string]string)
	for _, line := range strings.Split(m[1], "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			meta[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return meta, src[len(m[0]):]
}

var h1Re = regexp.MustCompile(`(?m)^# (.+)$`)

// extractTitle returns the page title from frontmatter or the first h1.
func extractTitle(body string, meta map[string]string) string {
	if t, ok := meta["title"]; ok && t != "" {
		return t
	}
	if m := h1Re.FindStringSubmatch(body); m != nil {
		return m[1]
	}
	return ""
}

var tagRe = regexp.MustCompile(`<[^>]+>`)

// plainText strips HTML tags, used for building the search index.
func plainText(html string) string {
	t := tagRe.ReplaceAllString(html, " ")
	return strings.Join(strings.Fields(t), " ")
}
