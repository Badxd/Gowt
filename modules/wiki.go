package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// .wiki <term> — search Wikipedia

type wikiResponse struct {
	Query struct {
		Pages map[string]struct {
			Title   string `json:"title"`
			Extract string `json:"extract"`
			Missing string `json:"missing"` // present (empty string) when page not found
		} `json:"pages"`
	} `json:"query"`
}

func wikiHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "📚 Usage: `.wiki <search term>`\nExample: `.wiki Elon Musk`")
	}
	query := strings.Join(args, " ")
	encoded := url.QueryEscape(query)

	apiURL := fmt.Sprintf(
		"https://en.wikipedia.org/w/api.php?action=query&titles=%s"+
			"&prop=extracts&exintro=true&explaintext=true"+
			"&format=json&redirects=1",
		encoded,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return ReplyErr(v, "❌ Request error: "+err.Error())
	}
	req.Header.Set("User-Agent", "PBX-Elite/1.0 (WhatsApp userbot)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ReplyErr(v, "❌ Network error — please try again.")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ReplyErr(v, "❌ Read error")
	}

	var data wikiResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return ReplyErr(v, "❌ Parse error — Wikipedia API may be down.")
	}

	for key, page := range data.Query.Pages {
		// Wikipedia uses key "-1" for pages that do not exist
		if key == "-1" {
			return ReplyErr(v, fmt.Sprintf(
				"❌ No Wikipedia article found for: *%s*\n"+
					"Try a different spelling or more specific term.",
				query,
			))
		}

		if strings.TrimSpace(page.Extract) == "" {
			return ReplyErr(v, fmt.Sprintf(
				"⚠️ Found *%s* but no summary available.\n"+
					"🔗 https://en.wikipedia.org/wiki/%s",
				page.Title,
				url.PathEscape(strings.ReplaceAll(page.Title, " ", "_")),
			))
		}

		// Truncate to 900 chars cleanly at sentence boundary
		extract := page.Extract
		if len(extract) > 900 {
			cut := extract[:900]
			if idx := strings.LastIndex(cut, ". "); idx > 400 {
				cut = cut[:idx+1]
			}
			extract = cut + " ..."
		}

		msg := fmt.Sprintf(
			"📚 *%s*\n\n%s\n\n🔗 https://en.wikipedia.org/wiki/%s",
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
