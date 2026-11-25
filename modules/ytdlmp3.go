package modules

import (
	"go.mau.fi/whatsmeow/types/events"
)

// .mp3 / .mp4 — YouTube download (requires yt-dlp — not available on Heroku)
// These commands show a helpful error explaining how to add yt-dlp.

func mp3Handler(v *events.Message, args []string) error {
	return ReplyErr(v, ytdlHerokuMsg)
}

func mp4Handler(v *events.Message, args []string) error {
	return ReplyErr(v, ytdlHerokuMsg)
}

func init() {
	Register(ModuleInfo{
		Name:        "Media Download",
		Description: "Download YouTube as MP3/MP4 (requires yt-dlp — not available on Heroku by default)",
		Commands: []CommandInfo{
			{Pattern: "mp3", Handler: mp3Handler, Sudo: true},
			{Pattern: "mp4", Handler: mp4Handler, Sudo: true},
		},
	})
}
