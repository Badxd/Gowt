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

// .lyrics <song name> — fetch lyrics via lyrics.ovh (free, no API key)

type lyricsResponse struct {
	Lyrics string `json:"lyrics"`
	Error  string `json:"error"`
}

func fetchLyrics(artist, title string) (string, error) {
	endpoint := fmt.Sprintf(
		"https://api.lyrics.ovh/v1/%s/%s",
		url.PathEscape(artist),
		url.PathEscape(title),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var res lyricsResponse
	json.Unmarshal(data, &res)

	if res.Error != "" {
		return "", fmt.Errorf(res.Error)
	}
	if res.Lyrics == "" {
		return "", fmt.Errorf("no lyrics found")
	}
	return strings.TrimSpace(res.Lyrics), nil
}

// .lyrics <artist> - <title>
// OR .lyrics <title>  (treats as title, artist as empty)
func lyricsHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .lyrics <artist> - <song title>\nExample: .lyrics Ed Sheeran - Shape of You")
	}

	full := strings.Join(args, " ")
	var artist, title string

	if idx := strings.Index(full, " - "); idx != -1 {
		artist = strings.TrimSpace(full[:idx])
		title = strings.TrimSpace(full[idx+3:])
	} else {
		// Try with blank artist
		parts := strings.Fields(full)
		if len(parts) >= 2 {
			artist = parts[0]
			title = strings.Join(parts[1:], " ")
		} else {
			artist = full
			title = full
		}
	}

	SendText(v, "🎵 Searching lyrics for: "+full)

	lyrics, err := fetchLyrics(artist, title)
	if err != nil {
		// Try swapping artist and title
		lyrics2, err2 := fetchLyrics(title, artist)
		if err2 != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Lyrics not found for: %s\nTry: .lyrics Artist - Song Title", full))
		}
		lyrics = lyrics2
	}

	// Truncate if too long
	if len(lyrics) > 3000 {
		lyrics = lyrics[:3000] + "\n\n...(truncated)"
	}

	return ReplyErr(v, fmt.Sprintf("🎶 *%s - %s*\n\n%s", artist, title, lyrics))
}

func init() {
	Register(ModuleInfo{
		Name:        "Lyrics",
		Description: "Fetch song lyrics",
		Commands: []CommandInfo{
			{Pattern: "lyrics", Handler: lyricsHandler, Sudo: true},
		},
	})
}
