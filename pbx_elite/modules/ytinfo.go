package modules

import (
	"fmt"
	"os/exec"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

// .ytinfo <url> — detailed info about a YouTube video using yt-dlp
// Separate from ytdl.go which handles download; this is info-only.

func ytinfoHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .ytinfo <youtube url>\nShows title, duration, views, uploader, description.")
	}

	videoURL := args[0]

	// Use yt-dlp to get video metadata as JSON
	cmd := exec.Command("yt-dlp",
		"--no-playlist",
		"--print", "%(title)s",
		"--print", "%(duration_string)s",
		"--print", "%(uploader)s",
		"--print", "%(view_count)s",
		"--print", "%(like_count)s",
		"--print", "%(upload_date)s",
		videoURL,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ReplyErr(v, "❌ yt-dlp error: "+string(out))
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 6 {
		return ReplyErr(v, "❌ Could not parse video info.")
	}

	title := lines[0]
	duration := lines[1]
	uploader := lines[2]
	views := lines[3]
	likes := lines[4]
	date := lines[5]

	// Format date YYYYMMDD → YYYY-MM-DD
	if len(date) == 8 {
		date = date[:4] + "-" + date[4:6] + "-" + date[6:]
	}

	// Format view count
	viewsFmt := formatCount(views)
	likesFmt := formatCount(likes)

	msg := fmt.Sprintf(`🎬 *YouTube Video Info*
━━━━━━━━━━━━━━━━━━
📌 *Title:* %s
👤 *Uploader:* %s
⏱️ *Duration:* %s
👁️ *Views:* %s
👍 *Likes:* %s
📅 *Uploaded:* %s
🔗 %s`,
		title, uploader, duration, viewsFmt, likesFmt, date, videoURL,
	)
	return ReplyErr(v, msg)
}

func formatCount(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "NA" || s == "None" {
		return "N/A"
	}
	// Parse and abbreviate
	var n int64
	fmt.Sscanf(s, "%d", &n)
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return s
	}
}

func init() {
	Register(ModuleInfo{
		Name:        "YTInfo",
		Description: "Get detailed YouTube video info (requires yt-dlp)",
		Commands: []CommandInfo{
			{Pattern: "ytinfo", Handler: ytinfoHandler, Sudo: true},
		},
	})
}
