package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.mau.fi/whatsmeow/types/events"
)

type redditPost struct {
	Data struct {
		Children []struct {
			Data struct {
				Title string `json:"title"`
				URL   string `json:"url"`
				Ups   int    `json:"ups"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func fetchMeme(subreddit string) (title, imgURL string, ups int, err error) {
	apiURL := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=20", subreddit)
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "PBXEliteBot/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var data redditPost
	if err = json.Unmarshal(body, &data); err != nil {
		return
	}

	for _, child := range data.Data.Children {
		u := child.Data.URL
		if len(u) > 4 {
			ext := u[len(u)-4:]
			if ext == ".jpg" || ext == ".png" || ext == "jpeg" {
				return child.Data.Title, u, child.Data.Ups, nil
			}
		}
	}
	err = fmt.Errorf("no image post found")
	return
}

func memeHandler(v *events.Message, args []string) error {
	title, imgURL, ups, err := fetchMeme("memes")
	if err != nil {
		return ReplyErr(v, "❌ Failed to fetch meme: "+err.Error())
	}
	msg := fmt.Sprintf("😂 *%s*\n👍 %d\n🔗 %s", title, ups, imgURL)
	return ReplyErr(v, msg)
}

func dankmemeHandler(v *events.Message, args []string) error {
	title, imgURL, ups, err := fetchMeme("dankmemes")
	if err != nil {
		return ReplyErr(v, "❌ Failed to fetch meme: "+err.Error())
	}
	msg := fmt.Sprintf("🔥 *%s*\n👍 %d\n🔗 %s", title, ups, imgURL)
	return ReplyErr(v, msg)
}

func init() {
	Register(ModuleInfo{
		Name:        "Meme",
		Description: "Fetch random memes from Reddit",
		Commands: []CommandInfo{
			{Pattern: "meme", Handler: memeHandler, Sudo: true},
			{Pattern: "dankmeme", Handler: dankmemeHandler, Sudo: true},
		},
	})
}
