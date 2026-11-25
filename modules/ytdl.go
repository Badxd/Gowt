package modules

import (
	"go.mau.fi/whatsmeow/types/events"
)

// .ytdl — Download YouTube audio (alias stub)
// .song and .mp3 are handled by download.go (which shows Heroku error if yt-dlp missing)
// This file only registers .ytdl to avoid duplicate symbol errors.

func ytdlHandler(v *events.Message, args []string) error {
	const ytdlMsg = `⚠️ *YouTube download requires yt-dlp.*

yt-dlp is not available on Heroku by default.

💡 *To enable yt-dlp on Heroku:*
1. Add buildpack: heroku buildpacks:add https://github.com/heroku/heroku-buildpack-apt
2. Create Aptfile with: python3
3. Install in .profile: pip3 install yt-dlp

Or host on a VPS where yt-dlp can be installed directly.`
	return ReplyErr(v, ytdlMsg)
}

func init() {
	Register(ModuleInfo{
		Name:        "YouTube Download",
		Description: "Download YouTube audio via yt-dlp (.ytdl alias)",
		Commands: []CommandInfo{
			{Pattern: "ytdl", Handler: ytdlHandler, Sudo: true},
		},
	})
}
