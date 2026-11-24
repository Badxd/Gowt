package modules

import (
	"go.mau.fi/whatsmeow/types/events"
)

// .song / .ytdl / .mp3 — Download YouTube audio
// Not available on Heroku (yt-dlp not installed).
// Shows helpful error with buildpack instructions.

const ytdlHerokuMsg = `⚠️ *YouTube download requires yt-dlp.*

yt-dlp is not available on Heroku by default.

💡 *To add yt-dlp to Heroku:*
` + "```" + `
heroku buildpacks:add https://github.com/heroku/heroku-buildpack-apt
` + "```" + `
Then create a file called ` + "`" + `Aptfile` + "`" + ` in your project root:
` + "```" + `
python3
python3-pip
` + "```" + `
And a ` + "`" + `.profile` + "`" + ` file:
` + "```" + `
pip3 install yt-dlp
` + "```" + `
Redeploy the bot after adding these.

Alternatively, host the bot on a VPS where you can install yt-dlp directly.`

func ytdlHandler(v *events.Message, args []string) error {
	return ReplyErr(v, ytdlHerokuMsg)
}

func songHandler(v *events.Message, args []string) error {
	return ReplyErr(v, ytdlHerokuMsg)
}

func init() {
	Register(ModuleInfo{
		Name:        "YouTube Download",
		Description: "Download YouTube audio (requires yt-dlp — not available on default Heroku)",
		Commands: []CommandInfo{
			{Pattern: "ytdl", Handler: ytdlHandler, Sudo: true},
			{Pattern: "song", Handler: songHandler, Sudo: true},
			{Pattern: "mp3", Handler: songHandler, Sudo: true},
		},
	})
}
