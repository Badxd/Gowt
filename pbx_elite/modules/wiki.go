package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

type wikiResponse struct {
	Query struct {
		Pages map[string]struct {
			Title   string `json:"title"`
			Extract string `json:"extract"`
		} `json:"pages"`
	} `json:"query"`
}

func wikiHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .wiki <search term>")
	}
	query := strings.Join(args, " ")
	encoded := url.QueryEscape(query)

	apiURL := fmt.Sprintf(
		"https://en.wikipedia.org/w/api.php?action=query&titles=%s&prop=extracts&exintro=true&explaintext=true&format=json&redirects=1",
		encoded,
	)

	resp, err := http.Get(apiURL)
	if err != nil {
		return ReplyErr(v, "❌ Request failed: "+err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ReplyErr(v, "❌ Read error")
	}

	var data wikiResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return ReplyErr(v, "❌ Parse error")
	}

	for _, page := range data.Query.Pages {
		if page.Extract == "" {
			return ReplyErr(v, fmt.Sprintf("❌ No Wikipedia article found for: *%s*", query))
		}

		// Truncate to 800 chars
		extract := page.Extract
		if len(extract) > 800 {
			extract = extract[:800] + "..."
		}

		msg := fmt.Sprintf("📚 *%s*\n\n%s\n\n🔗 https://en.wikipedia.org/wiki/%s",
			page.Title,
			extract,
			url.PathEscape(strings.ReplaceAll(page.Title, " ", "_")),
		)
		return ReplyErr(v, msg)
	}

	return ReplyErr(v, "❌ No results found.")
}

func init() {
	Register(ModuleInfo{
		Name:        "Wikipedia",
		Description: "Search Wikipedia for information",
		Commands: []CommandInfo{
			{Pattern: "wiki", Handler: wikiHandler, Sudo: true},
		},
	})
}
