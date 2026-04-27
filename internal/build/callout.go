package build

import (
	"regexp"
	"strings"
)

// calloutBlockRe matches a GitHub/Obsidian-style blockquote callout and its body.
// Group 1: callout type (e.g. NOTE, WARNING)
// Group 2: body lines (each starts with "> ")
var calloutBlockRe = regexp.MustCompile(`(?m)^> \[!(NOTE|TIP|WARNING|DANGER|INFO|SUCCESS|FAILURE|QUESTION|QUOTE|ABSTRACT|BUG|EXAMPLE)\]\n((?:> [^\n]*\n|>\n)*)`)

var calloutMeta = map[string][2]string{
	"NOTE":     {"note", "Note"},
	"TIP":      {"tip", "Tip"},
	"WARNING":  {"warning", "Warning"},
	"DANGER":   {"danger", "Danger"},
	"INFO":     {"info", "Info"},
	"SUCCESS":  {"success", "Success"},
	"FAILURE":  {"failure", "Failure"},
	"QUESTION": {"question", "Question"},
	"QUOTE":    {"quote", "Quote"},
	"ABSTRACT": {"abstract", "Abstract"},
	"BUG":      {"bug", "Bug"},
	"EXAMPLE":  {"example", "Example"},
}

// preprocessCallouts converts GitHub-style blockquote callouts to Material admonition HTML.
// Input:
//
//	> [!NOTE]
//	> body text
//
// Output:
//
//	<div class="admonition note"><p class="admonition-title">Note</p>...body...</div>
func preprocessCallouts(src string) string {
	return calloutBlockRe.ReplaceAllStringFunc(src, func(match string) string {
		typeMatch := regexp.MustCompile(`\[!(\w+)\]`).FindStringSubmatch(match)
		if typeMatch == nil {
			return match
		}
		key := typeMatch[1]
		meta, ok := calloutMeta[key]
		if !ok {
			meta = [2]string{strings.ToLower(key), key}
		}
		cssClass, title := meta[0], meta[1]

		// Collect body: strip leading "> " or ">" from each line after the header.
		lines := strings.Split(match, "\n")
		var bodyLines []string
		for _, line := range lines[1:] {
			if strings.HasPrefix(line, "> ") {
				bodyLines = append(bodyLines, line[2:])
			} else if line == ">" {
				bodyLines = append(bodyLines, "")
			}
		}
		body := strings.TrimSpace(strings.Join(bodyLines, "\n"))
		rendered := renderBodyMarkdown(body)

		return `<div class="admonition ` + cssClass + `">` +
			`<p class="admonition-title">` + title + `</p>` +
			rendered +
			`</div>`
	})
}
