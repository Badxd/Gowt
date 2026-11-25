package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

// .meme       — random meme from r/memes (image sent directly)
// .dankmeme   — random meme from r/dankmemes

type redditPost struct {
	Data struct {
		Children []struct {
			Data struct {
				Title   string `json:"title"`
				URL     string `json:"url"`
				Ups     int    `json:"ups"`
				IsVideo bool   `json:"is_video"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func fetchMeme(subreddit string) (title, imgURL string, ups int, err error) {
	apiURL := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=25", subreddit)
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "PBXEliteBot/2.0 (WhatsApp userbot)")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("Reddit returned HTTP %d", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// Reddit may return HTML on rate-limit
	if len(body) < 2 || body[0] != '{' {
		err = fmt.Errorf("Reddit returned non-JSON (may be rate-limited)")
		return
	}

	var data redditPost
	if err = json.Unmarshal(body, &data); err != nil {
		return
	}

	for _, child := range data.Data.Children {
		if child.Data.IsVideo {
			continue
		}
		u := child.Data.URL
		lower := strings.ToLower(u)
		if strings.HasSuffix(lower, ".jpg") ||
			strings.HasSuffix(lower, ".jpeg") ||
			strings.HasSuffix(lower, ".png") {
			return child.Data.Title, u, child.Data.Ups, nil
		}
		// i.redd.it without extension
		if strings.Contains(u, "i.redd.it") {
			return child.Data.Title, u, child.Data.Ups, nil
		}
	}
	err = fmt.Errorf("no image post found in top 25 posts")
	return
}

func sendMeme(v *events.Message, subreddit, emoji string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	title, imgURL, ups, err := fetchMeme(subreddit)
	if err != nil {
		return ReplyErr(v, "❌ Failed to fetch meme: "+err.Error())
	}

	// Download the image
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imgURL, nil)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("%s *%s*\n👍 %d\n🔗 %s", emoji, title, ups, imgURL))
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("%s *%s*\n👍 %d\n🔗 %s", emoji, title, ups, imgURL))
	}
	defer resp.Body.Close()

	imgData, err := io.ReadAll(resp.Body)
	if err != nil || len(imgData) == 0 {
		return ReplyErr(v, fmt.Sprintf("%s *%s*\n👍 %d\n🔗 %s", emoji, title, ups, imgURL))
	}

	// Upload and send as image
	uploaded, err := wac.Upload(context.Background(), imgData, whatsmeow.MediaImage)
	if err != nil {
		// Upload failed — just send URL
		return ReplyErr(v, fmt.Sprintf("%s *%s*\n👍 %d\n🔗 %s", emoji, title, ups, imgURL))
	}

	mime := "image/jpeg"
	lower := strings.ToLower(imgURL)
	if strings.HasSuffix(lower, ".png") {
		mime = "image/png"
	}

	caption := fmt.Sprintf("%s *%s*\n👍 %d upvotes", emoji, title, ups)
	fl := uploaded.FileLength

	_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			Caption:       &caption,
			Mimetype:      &mime,
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    &fl,
		},
	})
	return err
}

func memeHandler(v *events.Message, args []string) error {
	return sendMeme(v, "memes", "😂")
}

func dankmemeHandler(v *events.Message, args []string) error {
	return sendMeme(v, "dankmemes", "🔥")
}

func init() {
	Register(ModuleInfo{
		Name:        "Meme",
		Description: "Fetch and send random memes from Reddit",
		Commands: []CommandInfo{
			{Pattern: "meme", Handler: memeHandler, Sudo: true},
			{Pattern: "dankmeme", Handler: dankmemeHandler, Sudo: true},
		},
	})
}
