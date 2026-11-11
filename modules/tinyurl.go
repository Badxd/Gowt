package modules

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// .shorten <url>  — shorten URL via TinyURL API (free, no key)
// .unshorten <url> — expand a short URL to its destination

func shortenURL(longURL string) (string, error) {
	endpoint := "https://tinyurl.com/api-create.php?url=" + url.QueryEscape(longURL)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("TinyURL API returned %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	result := strings.TrimSpace(string(body))
	if !strings.HasPrefix(result, "http") {
		return "", fmt.Errorf("unexpected response: %s", result)
	}
	return result, nil
}

func unshortenURL(shortURL string) (string, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // stop at first redirect
		},
		Timeout: 15 * time.Second,
	}
	resp, err := client.Head(shortURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Follow all redirects manually to get final destination
	location := shortURL
	for i := 0; i < 10; i++ {
		resp2, err2 := client.Head(location)
		if err2 != nil {
			break
		}
		resp2.Body.Close()
		loc := resp2.Header.Get("Location")
		if loc == "" {
			break
		}
		location = loc
	}
	return location, nil
}

func shortenHandler(v *events.Message, args []string) error {
	var longURL string
	if len(args) > 0 {
		longURL = strings.TrimSpace(args[0])
	} else if q := getQuotedText(v); q != "" {
		longURL = strings.Fields(q)[0]
	}
	if longURL == "" {
		return ReplyErr(v, "Usage: .shorten <URL>")
	}
	if !strings.HasPrefix(longURL, "http") {
		longURL = "https://" + longURL
	}

	short, err := shortenURL(longURL)
	if err != nil {
		return ReplyErr(v, "❌ Could not shorten: "+err.Error())
	}
	return ReplyErr(v, "🔗 *Shortened:*\n"+short)
}

func unshortenHandler(v *events.Message, args []string) error {
	var shortURL string
	if len(args) > 0 {
		shortURL = strings.TrimSpace(args[0])
	} else if q := getQuotedText(v); q != "" {
		shortURL = strings.Fields(q)[0]
	}
	if shortURL == "" {
		return ReplyErr(v, "Usage: .unshorten <URL>")
	}
	if !strings.HasPrefix(shortURL, "http") {
		shortURL = "https://" + shortURL
	}

	dest, err := unshortenURL(shortURL)
	if err != nil {
		return ReplyErr(v, "❌ Could not unshorten: "+err.Error())
	}
	return ReplyErr(v, "🔓 *Expanded URL:*\n"+dest)
}

func init() {
	Register(ModuleInfo{
		Name:        "URLTools",
		Description: "Shorten and expand URLs",
		Commands: []CommandInfo{
			{Pattern: "shorten", Handler: shortenHandler, Sudo: true},
			{Pattern: "unshorten", Handler: unshortenHandler, Sudo: true},
		},
	})
}
