package modules

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"PbxGb/client"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// ytCheckInstalled — yt-dlp installed hai?
func ytCheckInstalled() bool {
	_, err := exec.LookPath("yt-dlp")
	return err == nil
}

// ffmpegInstalled — ffmpeg installed hai?
func ffmpegInstalled() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

// isYTURL — valid YouTube URL check
func isYTURL(u string) bool {
	return strings.Contains(u, "youtube.com") ||
		strings.Contains(u, "youtu.be")
}

// .song — YouTube se MP3 download
func songHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "❌ Use: .song https://youtube.com/watch?v=...\nOr: .song Never Gonna Give You Up")
	}

	if !ytCheckInstalled() {
		return ReplyErr(v, "❌ yt-dlp is not installed on the server.\nAdmin: `pip install yt-dlp`")
	}

	query := strings.Join(args, " ")
	_ = ReplyErr(v, "⏳ *Downloading audio...* Please wait")

	// URL ya search
	var ytURL string
	if isYTURL(query) {
		ytURL = query
	} else {
		ytURL = fmt.Sprintf("ytsearch1:%s", query)
	}

	tmpFile := fmt.Sprintf("/tmp/pbxgb_song_%s.mp3", v.Info.ID)
	defer os.Remove(tmpFile)

	cmd := exec.Command("yt-dlp",
		"-x", "--audio-format", "mp3",
		"--audio-quality", "128K",
		"--output", tmpFile,
		"--no-playlist",
		"--max-filesize", "15m",
		ytURL,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Download failed:\n`%s`", string(out)[:min(len(string(out)), 200)]))
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		return ReplyErr(v, "❌ File read failed.")
	}

	wac := client.Get()
	uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaAudio)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Upload failed: %v", err))
	}

	mime := strPtr("audio/mpeg")
	ptt := false
	flen := uint64(len(data))

	_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			Mimetype:      mime,
			PTT:           &ptt,
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    &flen,
		},
	})
	return err
}

// .yt — YouTube se MP4 video download
func ytHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "❌ Use: .yt https://youtube.com/watch?v=...\nOr: .yt Gangnam Style")
	}

	if !ytCheckInstalled() {
		return ReplyErr(v, "❌ yt-dlp is not installed on the server.\nAdmin: `pip install yt-dlp`")
	}

	query := strings.Join(args, " ")
	_ = ReplyErr(v, "⏳ *Downloading video...* Please wait")

	var ytURL string
	if isYTURL(query) {
		ytURL = query
	} else {
		ytURL = fmt.Sprintf("ytsearch1:%s", query)
	}

	tmpFile := fmt.Sprintf("/tmp/pbxgb_video_%s.mp4", v.Info.ID)
	defer os.Remove(tmpFile)

	cmd := exec.Command("yt-dlp",
		"-f", "best[height<=480][ext=mp4]/best[ext=mp4]/best",
		"--output", tmpFile,
		"--no-playlist",
		"--max-filesize", "15m",
		ytURL,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Download failed:\n`%s`", string(out)[:min(len(string(out)), 200)]))
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		return ReplyErr(v, "❌ File read failed.")
	}

	wac := client.Get()
	uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaVideo)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Upload failed: %v", err))
	}

	mime := strPtr("video/mp4")
	flen := uint64(len(data))

	_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		VideoMessage: &waProto.VideoMessage{
			Mimetype:      mime,
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    &flen,
		},
	})
	return err
}

// .bass — MP3 audio pe bass boost lagao (reply karo audio pe)
func bassHandler(v *events.Message, args []string) error {
	if !ffmpegInstalled() {
		return ReplyErr(v, "❌ ffmpeg is not installed on the server.\nAdmin: `apt install ffmpeg`")
	}

	quoted := GetQuotedMessage(v)
	if quoted == nil || quoted.AudioMessage == nil {
		return ReplyErr(v, "❌ Reply to an audio or song message and then type .bass")
	}

	_ = ReplyErr(v, "🎵 *Bass boosting...* Please wait")

	wac := client.Get()
	data, err := wac.Download(context.Background(), quoted.AudioMessage)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
	}

	// Write to temp
	inFile := fmt.Sprintf("/tmp/pbxgb_bass_in_%s.mp3", v.Info.ID)
	outFile := fmt.Sprintf("/tmp/pbxgb_bass_out_%s.mp3", v.Info.ID)
	defer os.Remove(inFile)
	defer os.Remove(outFile)

	if err := os.WriteFile(inFile, data, 0644); err != nil {
		return ReplyErr(v, "❌ Temp file error.")
	}

	// ffmpeg bass boost: equalizer filter
	cmd := exec.Command("ffmpeg",
		"-i", inFile,
		"-af", "equalizer=f=60:width_type=o:width=2:g=10,equalizer=f=120:width_type=o:width=2:g=6",
		"-q:a", "0",
		outFile,
		"-y",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Bass boost failed:\n`%s`", string(out)[:min(len(string(out)), 200)]))
	}

	bassData, err := os.ReadFile(outFile)
	if err != nil {
		return ReplyErr(v, "❌ Output file read failed.")
	}

	uploaded, err := wac.Upload(context.Background(), bassData, whatsmeow.MediaAudio)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Upload failed: %v", err))
	}

	mime := strPtr("audio/mpeg")
	ptt := false
	flen := uint64(len(bassData))

	_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			Mimetype:      mime,
			PTT:           &ptt,
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    &flen,
		},
	})
	if err != nil {
		return err
	}
	return ReplyErr(v, "🔊 *Bass Boosted!* 🎶")
}

// .img — Internet se image search & send
func imgSearchHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "❌ Use: .img cute cat")
	}
	query := url.QueryEscape(strings.Join(args, " "))
	// Unsplash free API (no key needed for basic)
	apiURL := fmt.Sprintf("https://source.unsplash.com/800x600/?%s", query)

	resp, err := http.Get(apiURL)
	if err != nil {
		return ReplyErr(v, "❌ Image fetch failed.")
	}
	defer resp.Body.Close()

	imgData, err := io.ReadAll(resp.Body)
	if err != nil {
		return ReplyErr(v, "❌ Image read failed.")
	}

	wac := client.Get()
	uploaded, err := wac.Upload(context.Background(), imgData, whatsmeow.MediaImage)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Upload failed: %v", err))
	}

	mime := strPtr("image/jpeg")
	flen := uint64(len(imgData))
	caption := fmt.Sprintf("🔍 *%s*", strings.Join(args, " "))

	_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			Caption:       &caption,
			Mimetype:      mime,
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    &flen,
		},
	})
	return err
}

func init() {
	Register(ModuleInfo{
		Name:        "Download",
		Description: "YouTube song/video download + bass boost + image search",
		Commands: []CommandInfo{
			{Pattern: "song",  Handler: songHandler,      Sudo: true},
			{Pattern: "ytmp3", Handler: songHandler,      Sudo: true},
			{Pattern: "yt",    Handler: ytHandler,        Sudo: true},
			{Pattern: "bass",  Handler: bassHandler,      Sudo: true},
			{Pattern: "img",   Handler: imgSearchHandler, Sudo: true},
		},
	})
}
