package modules

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"PbxGb/client"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

// ytInfo uses yt-dlp to get video info (title, duration)
func ytInfo(videoURL string) (title, duration string, err error) {
	cmd := exec.Command("yt-dlp",
		"--get-title", "--get-duration",
		"--no-playlist",
		videoURL,
	)
	out, err := cmd.Output()
	if err != nil {
		return "", "", err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) >= 2 {
		return lines[0], lines[1], nil
	}
	if len(lines) == 1 {
		return lines[0], "unknown", nil
	}
	return "unknown", "unknown", nil
}

// .yts <query> — search YouTube and return top result URL
func ytSearchHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .yts <search query>")
	}
	query := strings.Join(args, " ")
	cmd := exec.Command("yt-dlp",
		"--get-id", "--get-title",
		"--default-search", "ytsearch1",
		"--no-playlist",
		"ytsearch1:"+query,
	)
	out, err := cmd.Output()
	if err != nil {
		return ReplyErr(v, "❌ Search failed. Is yt-dlp installed?")
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return ReplyErr(v, "❌ No results found.")
	}
	title := lines[0]
	videoID := lines[1]
	link := "https://youtu.be/" + videoID

	return ReplyErr(v, fmt.Sprintf("🎬 *%s*\n🔗 %s", title, link))
}

// .ytdl <url> — download and send audio (mp3)
func ytdlHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .ytdl <youtube url>")
	}
	videoURL := args[0]
	if !strings.Contains(videoURL, "youtube.com") && !strings.Contains(videoURL, "youtu.be") {
		return ReplyErr(v, "❌ Please provide a valid YouTube URL.")
	}

	ReplyErr(v, "⬇️ Downloading audio, please wait...")

	title, dur, _ := ytInfo(videoURL)

	// Download audio as mp3 to /tmp
	outFile := "/tmp/yt_audio.%(ext)s"
	cmd := exec.Command("yt-dlp",
		"-x", "--audio-format", "mp3",
		"--audio-quality", "128K",
		"-o", outFile,
		"--no-playlist",
		videoURL,
	)
	if err := cmd.Run(); err != nil {
		return ReplyErr(v, "❌ Download failed: "+err.Error())
	}

	// Read the file
	audioData, err := readFile("/tmp/yt_audio.mp3")
	if err != nil {
		return ReplyErr(v, "❌ Could not read downloaded file.")
	}

	wac := client.Get()
	resp, err := wac.Upload(context.Background(), audioData, 4) // 4 = Audio
	if err != nil {
		return ReplyErr(v, "❌ Upload failed: "+err.Error())
	}

	caption := fmt.Sprintf("🎵 *%s*\n⏱️ %s", title, dur)
	mimetype := "audio/mpeg"
	filename := title + ".mp3"
	fl := uint64(len(audioData))

	_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			URL:           &resp.URL,
			DirectPath:    &resp.DirectPath,
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    &fl,
			Mimetype:      &mimetype,
			PTT:           boolPtrYT(false),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:      &v.Info.ID,
				Participant:   strPtr(v.Info.Sender.String()),
				QuotedMessage: v.Message,
			},
		},
	})
	if err != nil {
		// Fallback: send as document
		return ReplyErr(v, fmt.Sprintf("⚠️ Audio send failed, caption:\n%s\nFile: %s", caption, filename))
	}
	_ = caption
	_ = mimetype
	_ = filename
	return nil
}

func boolPtrYT(b bool) *bool { return &b }

// readFile reads a file from disk
func readFile(path string) ([]byte, error) {
	return exec.Command("cat", path).Output()
}

func init() {
	Register(ModuleInfo{
		Name:        "YouTube",
		Description: "YouTube search and audio download (requires yt-dlp)",
		Commands: []CommandInfo{
			{Pattern: "yts", Handler: ytSearchHandler, Sudo: true},
			{Pattern: "ytdl", Handler: ytdlHandler, Sudo: true},
		},
	})
}
