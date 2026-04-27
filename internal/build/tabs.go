package build

import (
	"fmt"
	"html/template"
	"regexp"
	"strings"
)

var tabHeaderRe = regexp.MustCompile(`^=== "([^"]+)"$`)

type tab struct {
	name    string
	content string
}

// preprocessTabs converts MkDocs-style content tabs to Material tabbed HTML.
// Input:
//
//	=== "Tab One"
//	    Content for tab one.
//
//	=== "Tab Two"
//	    Content for tab two.
//
// Output: Material tabbed-set HTML.
func preprocessTabs(src string) string {
	lines := strings.Split(src, "\n")
	var out []string
	var currentTabs []tab
	var currentTab *tab
	setCounter := 0

	flush := func() {
		if currentTab != nil {
			currentTabs = append(currentTabs, *currentTab)
			currentTab = nil
		}
		if len(currentTabs) > 0 {
			setCounter++
			out = append(out, buildTabSetHTML(currentTabs, setCounter))
			currentTabs = nil
		}
	}

	for _, line := range lines {
		if m := tabHeaderRe.FindStringSubmatch(line); m != nil {
			if currentTab != nil {
				currentTabs = append(currentTabs, *currentTab)
			}
			currentTab = &tab{name: m[1]}
			continue
		}

		if currentTab != nil {
			if strings.HasPrefix(line, "    ") {
				currentTab.content += line[4:] + "\n"
				continue
			}
			if strings.TrimSpace(line) == "" {
				// blank line: buffer it but keep collecting if next line is indented
				currentTab.content += "\n"
				continue
			}
			// non-indented non-blank line closes the tab set
			flush()
		}

		out = append(out, line)
	}

	flush()
	return strings.Join(out, "\n")
}

func buildTabSetHTML(tabs []tab, setID int) string {
	var sb strings.Builder
	n := len(tabs)
	sb.WriteString(fmt.Sprintf(`<div class="tabbed-set tabbed-alternate" data-tabs="__tabbed_%d:%d">`, setID, n))

	for i := range tabs {
		checked := ""
		if i == 0 {
			checked = ` checked="checked"`
		}
		sb.WriteString(fmt.Sprintf(`<input%s id="__tabbed_%d_%d" name="__tabbed_%d" type="radio">`, checked, setID, i+1, setID))
	}

	sb.WriteString(`<div class="tabbed-content">`)
	for _, t := range tabs {
		sb.WriteString(`<div class="tabbed-block">`)
		sb.WriteString(renderBodyMarkdown(strings.TrimSpace(t.content)))
		sb.WriteString(`</div>`)
	}
	sb.WriteString(`</div>`)

	sb.WriteString(`<div class="tabbed-labels">`)
	for i, t := range tabs {
		sb.WriteString(fmt.Sprintf(`<label for="__tabbed_%d_%d">%s</label>`, setID, i+1, template.HTMLEscapeString(t.name)))
	}
	sb.WriteString(`</div>`)
	sb.WriteString(`</div>`)
	return sb.String()
}
