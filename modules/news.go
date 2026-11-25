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

// .news [query] — fetch top news headlines via GNews free API
// or gnews.io (free tier: 100 req/day)

const gnewsAPIKey = "" // leave empty for RSS fallback

type gnewsResponse struct {
	Articles []struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		URL         string `json:"url"`
		Source      struct {
			Name string `json:"name"`
		} `json:"source"`
		PublishedAt string `json:"publishedAt"`
	} `json:"articles"`
}

// fetchNewsGnews tries gnews.io API if key is set
func fetchNewsGnews(query string) (string, error) {
	params := url.Values{}
	params.Set("lang", "en")
	params.Set("max", "5")
	params.Set("apikey", gnewsAPIKey)
	if query != "" {
		params.Set("q", query)
	} else {
		params.Set("topic", "breaking-news")
	}

	endpoint := "https://gnews.io/api/v4/search?" + params.Encode()
	if query == "" {
		endpoint = "https://gnews.io/api/v4/top-headlines?" + params.Encode()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var result gnewsResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	if len(result.Articles) == 0 {
		return "", fmt.Errorf("no articles found")
	}

	var sb strings.Builder
	for i, a := range result.Articles {
		if i >= 5 {
			break
		}
		sb.WriteString(fmt.Sprintf("*%d. %s*\n_%s_\n🔗 %s\n\n",
			i+1, a.Title, a.Source.Name, a.URL))
	}
	return strings.TrimSpace(sb.String()), nil
}

// fetchNewsBBC fallback: BBC RSS
func fetchNewsBBC(query string) (string, error) {
	rssFeed := "https://feeds.bbci.co.uk/news/rss.xml"
	if query != "" {
		// no query support in RSS, just return top
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, rssFeed, nil)
	req.Header.Set("User-Agent", "PBX-Elite/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	content := string(data)

	// Simple XML parse for titles
	var headlines []string
	for i := 0; i < 5; i++ {
		start := strings.Index(content, "<title>")
		if start == -1 {
			break
		}
		content = content[start+7:]
		end := strings.Index(content, "</title>")
		if end == -1 {
			break
		}
		title := strings.TrimSpace(content[:end])
		title = strings.ReplaceAll(title, "<![CDATA[", "")
		title = strings.ReplaceAll(title, "]]>", "")
		if title != "" && title != "BBC News" {
			headlines = append(headlines, title)
		}
		content = content[end:]
	}

	if len(headlines) == 0 {
		return "", fmt.Errorf("no headlines found")
	}

	var sb strings.Builder
	sb.WriteString("📰 *BBC Top Headlines:*\n\n")
	for i, h := range headlines {
		sb.WriteString(fmt.Sprintf("*%d.* %s\n", i+1, h))
	}
	return strings.TrimSpace(sb.String()), nil
}

// .news [query]
func newsHandler(v *events.Message, args []string) error {
	query := strings.Join(args, " ")

	var result string
	var err error

	if gnewsAPIKey != "" {
		result, err = fetchNewsGnews(query)
	} else {
		result, err = fetchNewsBBC(query)
	}

	if err != nil {
		return ReplyErr(v, "❌ Could not fetch news: "+err.Error())
	}

	header := "📰 *Latest News*"
	if query != "" {
		header = fmt.Sprintf("📰 *News: %s*", query)
	}

	return ReplyErr(v, header+"\n━━━━━━━━━━━━━━━━\n"+result)
}

func init() {
	Register(ModuleInfo{
		Name:        "News",
		Description: "Fetch latest news headlines",
		Commands: []CommandInfo{
			{Pattern: "news", Handler: newsHandler, Sudo: true},
		},
	})
}
