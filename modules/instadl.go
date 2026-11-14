package modules

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// .igdl  <url>  — download Instagram/TikTok/Facebook/Twitter media via yt-dlp
// .fbdl  <url>  — alias for Facebook specifically
// .tikdl <url>  — alias for TikTok specifically
// All use yt-dlp under the hood; best quality ≤50 MB

func socialDownload(socialURL string) ([]byte, string, error) {
	dir, err := os.MkdirTemp("", "pbx_social_")
	if err != nil {
		return nil, "", err
	}
	defer os.RemoveAll(dir)

	outTmpl := filepath.Join(dir, "media.%(ext)s")

	// prefer mp4; fall back to best available
	args := []string{
		"-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/mp4/best",
		"--merge-output-format", "mp4",
		"--max-filesize", "49m",
		"--no-playlist",
		"-o", outTmpl,
		"--quiet",
		socialURL,
	}
	cmd := exec.Command("yt-dlp", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		// try audio-only fallback (stories / audio reels)
		audioArgs := []string{
			"-x", "--audio-format", "mp3",
			"--max-filesize", "49m",
			"--no-playlist",
			"-o", outTmpl,
			"--quiet",
			socialURL,
		}
		cmd2 := exec.Command("yt-dlp", audioArgs...)
		if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
			return nil, "", fmt.Errorf("yt-dlp failed: %s | audio fallback: %s", string(out), string(out2))
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		return nil, "", fmt.Errorf("no output file found after download")
	}

	filePath := filepath.Join(dir, entries[0].Name())
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", err
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	return data, ext, nil
}

func sendSocialMedia(v *events.Message, data []byte, ext string, caption string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	switch ext {
	case "mp4", "mov", "webm":
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaVideo)
		if err != nil {
			return err
		}
		msg := &waE2E.Message{
			VideoMessage: &waE2E.VideoMessage{
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
				Mimetype:      strPtr("video/mp4"),
				Caption:       strPtr(caption),
			},
		}
		_, err = wac.SendMessage(context.Background(), v.Info.Chat, msg)
		return err

	case "mp3", "m4a", "ogg", "aac":
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaAudio)
		if err != nil {
			return err
		}
		msg := &waE2E.Message{
			AudioMessage: &waE2E.AudioMessage{
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
				Mimetype:      strPtr("audio/mpeg"),
			},
		}
		_, err = wac.SendMessage(context.Background(), v.Info.Chat, msg)
		return err

	default:
		// send as document
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaDocument)
		if err != nil {
			return err
		}
		mime := "application/octet-stream"
		fname := "media." + ext
		msg := &waE2E.Message{
			DocumentMessage: &waE2E.DocumentMessage{
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
				Mimetype:      &mime,
				FileName:      &fname,
				Caption:       strPtr(caption),
			},
		}
		_, err = wac.SendMessage(context.Background(), v.Info.Chat, msg)
		return err
	}
}

func igdlHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .igdl <instagram/tiktok/twitter/fb url>")
	}
	url := args[0]
	ReplyErr(v, "⬇️ Downloading... please wait")

	start := time.Now()
	data, ext, err := socialDownload(url)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
	}

	elapsed := time.Since(start).Round(time.Second)
	caption := fmt.Sprintf("Downloaded in %s via yt-dlp", elapsed)
	if err := sendSocialMedia(v, data, ext, caption); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Send failed: %v", err))
	}
	return nil
}

func fbdlHandler(v *events.Message, args []string) error {
	return igdlHandler(v, args)
}

func tikdlHandler(v *events.Message, args []string) error {
	return igdlHandler(v, args)
}

func init() {
	Register(ModuleInfo{
		Name:        "Social Downloader",
		Description: "Download media from Instagram, TikTok, Facebook, Twitter via yt-dlp",
		Commands: []CommandInfo{
			{Pattern: "igdl", Handler: igdlHandler, Sudo: true},
			{Pattern: "fbdl", Handler: fbdlHandler, Sudo: true},
			{Pattern: "tikdl", Handler: tikdlHandler, Sudo: true},
		},
	})
}
