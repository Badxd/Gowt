package modules

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"PbxGb/client"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// .mp3 <youtube url>  — download YouTube audio as MP3 (via yt-dlp)
// .mp4 <youtube url>  — download YouTube video (low quality, ≤50MB via yt-dlp)
// separate from ytdl.go which handles generic downloads

func ytDownloadAudio(ytURL string) ([]byte, string, error) {
	dir, err := os.MkdirTemp("", "pbx_mp3_")
	if err != nil {
		return nil, "", err
	}
	defer os.RemoveAll(dir)

	outTmpl := filepath.Join(dir, "audio.%(ext)s")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx,
		"yt-dlp",
		"--no-playlist",
		"-x",                    // extract audio
		"--audio-format", "mp3", // convert to mp3
		"--audio-quality", "5",  // ~128kbps
		"--max-filesize", "49m",
		"-o", outTmpl,
		ytURL,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, "", fmt.Errorf("yt-dlp: %s", strings.TrimSpace(string(out)))
	}

	// Find the output file
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".mp3") {
			path := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(path)
			return data, e.Name(), err
		}
	}
	return nil, "", fmt.Errorf("audio file not found after download")
}

func ytDownloadVideo(ytURL string) ([]byte, string, error) {
	dir, err := os.MkdirTemp("", "pbx_mp4_")
	if err != nil {
		return nil, "", err
	}
	defer os.RemoveAll(dir)

	outTmpl := filepath.Join(dir, "video.%(ext)s")

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx,
		"yt-dlp",
		"--no-playlist",
		"-f", "bestvideo[height<=480][ext=mp4]+bestaudio[ext=m4a]/mp4/best[height<=480]",
		"--merge-output-format", "mp4",
		"--max-filesize", "49m",
		"-o", outTmpl,
		ytURL,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, "", fmt.Errorf("yt-dlp: %s", strings.TrimSpace(string(out)))
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".mp4") {
			path := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(path)
			return data, e.Name(), err
		}
	}
	return nil, "", fmt.Errorf("video file not found after download")
}

func mp3Handler(v *events.Message, args []string) error {
	wac := client.Get()

	var ytURL string
	if len(args) > 0 {
		ytURL = args[0]
	} else if q := getQuotedText(v); q != "" {
		ytURL = strings.Fields(q)[0]
	}
	if ytURL == "" {
		return ReplyErr(v, "Usage: .mp3 <YouTube URL>")
	}
	if !strings.Contains(ytURL, "youtube.com") && !strings.Contains(ytURL, "youtu.be") {
		return ReplyErr(v, "❌ Please provide a valid YouTube URL")
	}

	ReplyErr(v, "⏳ Downloading audio, please wait...")

	data, filename, err := ytDownloadAudio(ytURL)
	if err != nil {
		return ReplyErr(v, "❌ Download failed: "+err.Error())
	}

	uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaAudio)
	if err != nil {
		return ReplyErr(v, "❌ Upload failed: "+err.Error())
	}

	_ = filename
	size := uint32(len(data))
	duration := uint32(0) // unknown without ffprobe

	audioMsg := &waE2E.AudioMessage{
		URL:           &uploaded.URL,
		DirectPath:    &uploaded.DirectPath,
		MediaKey:      uploaded.MediaKey,
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    &uploaded.FileLength,
		Mimetype:      strPtr("audio/mpeg"),
		Seconds:       &duration,
		FileSize:      &size,
	}
	_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waE2E.Message{AudioMessage: audioMsg})
	return err
}

func mp4Handler(v *events.Message, args []string) error {
	wac := client.Get()

	var ytURL string
	if len(args) > 0 {
		ytURL = args[0]
	} else if q := getQuotedText(v); q != "" {
		ytURL = strings.Fields(q)[0]
	}
	if ytURL == "" {
		return ReplyErr(v, "Usage: .mp4 <YouTube URL>")
	}
	if !strings.Contains(ytURL, "youtube.com") && !strings.Contains(ytURL, "youtu.be") {
		return ReplyErr(v, "❌ Please provide a valid YouTube URL")
	}

	ReplyErr(v, "⏳ Downloading video (≤480p, ≤49MB), please wait...")

	data, _, err := ytDownloadVideo(ytURL)
	if err != nil {
		return ReplyErr(v, "❌ Download failed: "+err.Error())
	}

	uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaVideo)
	if err != nil {
		return ReplyErr(v, "❌ Upload failed: "+err.Error())
	}

	width := uint32(854)
	height := uint32(480)
	duration := uint32(0)

	videoMsg := &waE2E.VideoMessage{
		URL:           &uploaded.URL,
		DirectPath:    &uploaded.DirectPath,
		MediaKey:      uploaded.MediaKey,
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    &uploaded.FileLength,
		Mimetype:      strPtr("video/mp4"),
		Width:         &width,
		Height:        &height,
		Seconds:       &duration,
	}
	_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waE2E.Message{VideoMessage: videoMsg})
	return err
}

func init() {
	Register(ModuleInfo{
		Name:        "YTMedia",
		Description: "Download YouTube audio (.mp3) and video (.mp4)",
		Commands: []CommandInfo{
			{Pattern: "mp3", Handler: mp3Handler, Sudo: true},
			{Pattern: "mp4", Handler: mp4Handler, Sudo: true},
		},
	})
}
