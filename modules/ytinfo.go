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

// .ytinfo <url> — YouTube video info using oEmbed (no yt-dlp required, Heroku-safe)

type oEmbedResponse struct {
	Title        string `json:"title"`
	AuthorName   string `json:"author_name"`
	AuthorURL    string `json:"author_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	ProviderName string `json:"provider_name"`
}

func ytinfoHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "🎬 Usage: `.ytinfo <youtube url>`\nExample: `.ytinfo https://youtu.be/dQw4w9WgXcQ`")
	}
	videoURL := args[0]

	// Validate it looks like a YouTube URL
	if !strings.Contains(videoURL, "youtube.com") && !strings.Contains(videoURL, "youtu.be") {
		return ReplyErr(v, "❌ Please provide a valid YouTube URL.")
	}

	// Extract video ID for thumbnail
	videoID := extractYouTubeID(videoURL)

	// Use YouTube oEmbed API — works without yt-dlp, no API key required
	oembedURL := "https://www.youtube.com/oembed?url=" + url.QueryEscape(videoURL) + "&format=json"

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, oembedURL, nil)
	if err != nil {
		return ReplyErr(v, "❌ Request error: "+err.Error())
	}
	req.Header.Set("User-Agent", "PBX-Elite/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ReplyErr(v, "❌ Network error — check the URL and try again.")
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return ReplyErr(v, "❌ Video not found or is private.")
	}
	if resp.StatusCode != http.StatusOK {
		return ReplyErr(v, fmt.Sprintf("❌ YouTube API error (HTTP %d).", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ReplyErr(v, "❌ Read error.")
	}

	var info oEmbedResponse
	if err := json.Unmarshal(body, &info); err != nil {
		return ReplyErr(v, "❌ Parse error.")
	}

	thumb := ""
	if videoID != "" {
		thumb = fmt.Sprintf("🖼️ Thumbnail: https://img.youtube.com/vi/%s/hqdefault.jpg\n", videoID)
	}

	msg := fmt.Sprintf(
		"🎬 *YouTube Video Info*\n"+
			"━━━━━━━━━━━━━━━━━━\n"+
			"📌 *Title:* %s\n"+
			"👤 *Channel:* %s\n"+
			"%s"+
			"🔗 %s",
		info.Title,
		info.AuthorName,
		thumb,
		videoURL,
	)
	return ReplyErr(v, msg)
}

func extractYouTubeID(rawURL string) string {
	// youtu.be/<id>
	if strings.Contains(rawURL, "youtu.be/") {
		parts := strings.Split(rawURL, "youtu.be/")
		if len(parts) > 1 {
			id := strings.Split(parts[1], "?")[0]
			id = strings.Split(id, "&")[0]
			return id
		}
	}
	// youtube.com/watch?v=<id>
	if strings.Contains(rawURL, "v=") {
		parts := strings.Split(rawURL, "v=")
		if len(parts) > 1 {
			id := strings.Split(parts[1], "&")[0]
			return id
		}
	}
	return ""
}

func init() {
	Register(ModuleInfo{
		Name:        "YTInfo",
		Description: "Get YouTube video title and channel (Heroku-safe, no yt-dlp required)",
		Commands: []CommandInfo{
			{Pattern: "ytinfo", Handler: ytinfoHandler, Sudo: true},
			{Pattern: "yti", Handler: ytinfoHandler, Sudo: true},
		},
	})
}
