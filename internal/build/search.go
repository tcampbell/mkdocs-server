package build

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type searchDoc struct {
	Location string `json:"location"`
	Title    string `json:"title"`
	Text     string `json:"text"`
}

type searchIndex struct {
	Config searchConfig `json:"config"`
	Docs   []searchDoc  `json:"docs"`
}

type searchConfig struct {
	Lang            []string `json:"lang"`
	MinSearchLength int      `json:"min_search_length"`
	PrebuildIndex   bool     `json:"prebuild_index"`
	Separator       string   `json:"separator"`
}

func writeSearchIndex(siteDir string, docs []searchDoc) error {
	idx := searchIndex{
		Config: searchConfig{
			Lang:            []string{"en"},
			MinSearchLength: 3,
			PrebuildIndex:   false,
			Separator:       `[\s\-,:!=\[\]()\"/]+|(?!\b)(?=[A-Z][a-z])|(?<=[a-z\-])(?=[0-9])|(?=[0-9\-])(?<=[A-Z])`,
		},
		Docs: docs,
	}
	data, err := json.Marshal(idx)
	if err != nil {
		return err
	}
	dir := filepath.Join(siteDir, "search")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "search_index.json"), data, 0o644)
}
