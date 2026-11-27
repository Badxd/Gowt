package modules

import (
	"go.mau.fi/whatsmeow/types/events"
)

// .mp3 / .mp4 — YouTube download (requires yt-dlp — not available on Heroku)
// These commands show a helpful error explaining how to add yt-dlp.

const ytdlMediaMsg = `⚠️ *YouTube download requires yt-dlp.*

yt-dlp is not available on Heroku by default.

💡 *To enable yt-dlp on Heroku:*
1. Add buildpack: heroku buildpacks:add https://github.com/heroku/heroku-buildpack-apt
2. Create Aptfile with: python3
3. Install in .profile: pip3 install yt-dlp

Or host on a VPS where yt-dlp can be installed directly.`

func mp3Handler(v *events.Message, args []string) error {
	return ReplyErr(v, ytdlMediaMsg)
}

func mp4Handler(v *events.Message, args []string) error {
	return ReplyErr(v, ytdlMediaMsg)
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
