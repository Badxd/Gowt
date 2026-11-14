package modules

// ── MEDIA DOWNLOAD MODULE ────────────────────────────────────────────────────
// Commands:
//   .song   <YouTube URL or search query>   — audio MP3
//   .video  <YouTube URL or search query>   — video MP4
//   .fbdl   <Facebook URL>                  — Facebook video
//   .ttdl   <TikTok URL>                    — TikTok video (no watermark)
//   .igdl2  <Instagram URL>                 — Instagram photo / video / reel
//
// Architecture:
//   Primary  — HTTP download API (env: MEDIA_API_URL, MEDIA_API_KEY)
//   Fallback — yt-dlp binary (must be installed on server)
//
// Env vars (set in .env or deploy config vars):
//   MEDIA_API_URL   — base URL of your download API  (optional, has default)
//   MEDIA_API_KEY   — your API key                   (optional, has default)
//   MEDIA_API_KEY2  — optional backup API key

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"PbxGb/client"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

func init() {
	Register(ModuleInfo{
		Name:        "MediaDownload",
		Description: "Download songs, videos, FB, TikTok, Instagram via API + yt-dlp fallback",
		Commands: []CommandInfo{
			{Pattern: "song",    Handler: songHandlerMDL,  Sudo: true},
			{Pattern: "video",   Handler: videoHandler, Sudo: true},
			{Pattern: "fbdl",    Handler: fbdlHandler2, Sudo: true},
			{Pattern: "fbvideo", Handler: fbdlHandler2, Sudo: true},
			{Pattern: "ttdl",    Handler: ttdlHandler,  Sudo: true},
			{Pattern: "ttvideo", Handler: ttdlHandler,  Sudo: true},
			{Pattern: "igdl2",   Handler: igdl2Handler, Sudo: true},
		},
	})
}

// ── ENV CONFIG ───────────────────────────────────────────────────────────────

func mediaAPIBase() string {
	v := strings.TrimRight(os.Getenv("MEDIA_API_URL"), "/")
	if v == "" {
		return "https://knightbotapi.stream"
	}
	return v
}

func mediaAPIKey() string {
	k := os.Getenv("MEDIA_API_KEY")
	if k == "" {
		k = os.Getenv("MEDIA_API_KEY2")
	}
	if k == "" {
		k = "knight"
	}
	return k
}

// ── API RESPONSE SHAPES ──────────────────────────────────────────────────────

type mediaAPIResp struct {
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"`
	Error   string          `json:"error"`
	Message string          `json:"message"`
}

type mediaResult struct {
	Download    string `json:"download"`
	DlURL       string `json:"dl_url"`
	Dl          string `json:"dl"`
	DownloadUrl string `json:"downloadUrl"`
	URL         string `json:"url"`
	Audio       string `json:"audio"`
	Video       string `json:"video"`
	MP3         string `json:"mp3"`
	MP4         string `json:"mp4"`
	Title       string `json:"title"`
	Thumbnail   string `json:"thumbnail"`
	Image       string `json:"image"`
	Artist      string `json:"artist"`
	Duration    string `json:"duration"`
}

func (r mediaResult) bestURL() string {
	for _, u := range []string{r.Download, r.DlURL, r.Dl, r.DownloadUrl, r.Video, r.Audio, r.MP4, r.MP3, r.URL} {
		if strings.HasPrefix(u, "http") {
			return u
		}
	}
	return ""
}

// ── HTTP FETCH ───────────────────────────────────────────────────────────────

func apiGet(path string, params map[string]string) (mediaResult, error) {
	base := mediaAPIBase()
	key := mediaAPIKey()

	u, err := url.Parse(base + path)
	if err != nil {
		return mediaResult{}, err
	}
	q := u.Query()
	q.Set("apikey", key)
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	hc := &http.Client{Timeout: 90 * time.Second}
	resp, err := hc.Get(u.String())
	if err != nil {
		return mediaResult{}, fmt.Errorf("api request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mediaResult{}, fmt.Errorf("read body: %w", err)
	}

	var apiResp mediaAPIResp
	if err := json.Unmarshal(body, &apiResp); err != nil {
		var direct mediaResult
		if err2 := json.Unmarshal(body, &direct); err2 != nil {
			return mediaResult{}, fmt.Errorf("parse response: %w", err)
		}
		return direct, nil
	}

	if !apiResp.Success && apiResp.Error != "" {
		msg := apiResp.Error
		if apiResp.Message != "" {
			msg = apiResp.Message
		}
		return mediaResult{}, fmt.Errorf("api error: %s", msg)
	}

	if len(apiResp.Result) > 0 {
		var res mediaResult
		if err := json.Unmarshal(apiResp.Result, &res); err == nil && res.bestURL() != "" {
			return res, nil
		}
	}

	var direct mediaResult
	if err := json.Unmarshal(body, &direct); err != nil {
		return mediaResult{}, fmt.Errorf("parse result: %w", err)
	}
	return direct, nil
}

// downloadBytes fetches raw bytes from a URL
func downloadBytesMedia(dlURL string) ([]byte, error) {
	hc := &http.Client{Timeout: 3 * time.Minute}
	resp, err := hc.Get(dlURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// ── YT-DLP FALLBACK ──────────────────────────────────────────────────────────

func ytdlpAudio(inputURL string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "pbx_song_")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	out := filepath.Join(dir, "audio.%(ext)s")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--no-playlist", "-x", "--audio-format", "mp3",
		"--audio-quality", "5", "--max-filesize", "49m",
		"-o", out, inputURL,
	)
	if o, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("yt-dlp: %s", strings.TrimSpace(string(o)))
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		return os.ReadFile(filepath.Join(dir, e.Name()))
	}
	return nil, fmt.Errorf("no output file")
}

func ytdlpVideo(inputURL string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "pbx_video_")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	out := filepath.Join(dir, "video.%(ext)s")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--no-playlist",
		"-f", "bestvideo[height<=480][ext=mp4]+bestaudio[ext=m4a]/mp4/best[height<=480]",
		"--merge-output-format", "mp4",
		"--max-filesize", "49m",
		"-o", out, inputURL,
	)
	if o, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("yt-dlp: %s", strings.TrimSpace(string(o)))
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		return os.ReadFile(filepath.Join(dir, e.Name()))
	}
	return nil, fmt.Errorf("no output file")
}

// ── WHATSAPP SEND HELPERS ────────────────────────────────────────────────────

func sendAudio(v *events.Message, data []byte) error {
	wac := client.Get()
	uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaAudio)
	if err != nil {
		return ReplyErr(v, "❌ Upload failed: "+err.Error())
	}
	dur := uint32(0)
	ptt := false
	_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    &uploaded.FileLength,
			Mimetype:      strPtr("audio/mpeg"),
			Seconds:       &dur,
			PTT:           &ptt,
			ContextInfo: &waProto.ContextInfo{
				StanzaID:      &v.Info.ID,
				Participant:   strPtr(v.Info.Sender.String()),
				QuotedMessage: v.Message,
			},
		},
	})
	return err
}

func sendVideo(v *events.Message, data []byte, caption string) error {
	wac := client.Get()
	uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaVideo)
	if err != nil {
		return ReplyErr(v, "❌ Upload failed: "+err.Error())
	}
	w := uint32(854)
	h := uint32(480)
	dur := uint32(0)
	_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		VideoMessage: &waProto.VideoMessage{
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    &uploaded.FileLength,
			Mimetype:      strPtr("video/mp4"),
			Caption:       strPtr(caption),
			Width:         &w,
			Height:        &h,
			Seconds:       &dur,
			ContextInfo: &waProto.ContextInfo{
				StanzaID:      &v.Info.ID,
				Participant:   strPtr(v.Info.Sender.String()),
				QuotedMessage: v.Message,
			},
		},
	})
	return err
}

func sendImage(v *events.Message, data []byte, caption string) error {
	wac := client.Get()
	uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaImage)
	if err != nil {
		return ReplyErr(v, "❌ Upload failed: "+err.Error())
	}
	_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    &uploaded.FileLength,
			Mimetype:      strPtr("image/jpeg"),
			Caption:       strPtr(caption),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:      &v.Info.ID,
				Participant:   strPtr(v.Info.Sender.String()),
				QuotedMessage: v.Message,
			},
		},
	})
	return err
}

// ── .song ────────────────────────────────────────────────────────────────────
// Usage: .song <YouTube URL or search query>

func songHandlerMDL(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .song <YouTube URL or song name>\nExample: .song https://youtu.be/xxx\nExample: .song Diljit Dosanjh Born to Shine")
	}

	query := strings.Join(args, " ")
	Reply(v, "🎵 Downloading song, please wait...")

	res, apiErr := apiGet("/api/download/ytaudio", map[string]string{"url": query})
	if apiErr == nil && res.bestURL() != "" {
		data, err := downloadBytesMedia(res.bestURL())
		if err == nil && len(data) > 0 {
			title := res.Title
			if title == "" {
				title = query
			}
			caption := fmt.Sprintf("🎵 *%s*", title)
			if res.Artist != "" {
				caption += "\n👤 " + res.Artist
			}
			if res.Duration != "" {
				caption += "\n⏱️ " + res.Duration
			}
			Reply(v, caption)
			return sendAudio(v, data)
		}
	}

	Reply(v, "⏳ Trying alternate method...")
	data, err := ytdlpAudio(query)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Download failed.\nAPI: %v\nyt-dlp: %v", apiErr, err))
	}
	return sendAudio(v, data)
}

// ── .video ───────────────────────────────────────────────────────────────────
// Usage: .video <YouTube URL or search query>

func videoHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .video <YouTube URL or video name>\nExample: .video https://youtu.be/xxx")
	}

	query := strings.Join(args, " ")
	Reply(v, "🎬 Downloading video, please wait...")

	res, apiErr := apiGet("/api/download/ytvideo", map[string]string{"url": query})
	if apiErr == nil && res.bestURL() != "" {
		data, err := downloadBytesMedia(res.bestURL())
		if err == nil && len(data) > 0 {
			title := res.Title
			if title == "" {
				title = query
			}
			return sendVideo(v, data, "🎬 *"+title+"*")
		}
	}

	Reply(v, "⏳ Trying alternate method...")
	data, err := ytdlpVideo(query)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Download failed.\nAPI: %v\nyt-dlp: %v", apiErr, err))
	}
	return sendVideo(v, data, "🎬 "+query)
}

// ── .fbdl / .fbvideo ─────────────────────────────────────────────────────────
// Usage: .fbdl <Facebook URL>

func fbdlHandler2(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .fbdl <Facebook video URL>\nExample: .fbdl https://fb.watch/xxx")
	}

	fbURL := args[0]
	if !strings.Contains(fbURL, "facebook.com") && !strings.Contains(fbURL, "fb.watch") && !strings.Contains(fbURL, "fb.com") {
		return ReplyErr(v, "❌ Please provide a valid Facebook video URL.")
	}

	Reply(v, "📥 Downloading Facebook video...")

	res, apiErr := apiGet("/api/download/facebook", map[string]string{"url": fbURL})
	if apiErr == nil && res.bestURL() != "" {
		data, err := downloadBytesMedia(res.bestURL())
		if err == nil && len(data) > 0 {
			title := res.Title
			if title == "" {
				title = "Facebook Video"
			}
			return sendVideo(v, data, "📘 *"+title+"*")
		}
	}

	Reply(v, "⏳ Trying alternate method...")
	data, err := ytdlpVideo(fbURL)
	if err != nil || len(data) == 0 {
		return ReplyErr(v, fmt.Sprintf("❌ Facebook download failed.\nAPI: %v\nyt-dlp: %v", apiErr, err))
	}
	return sendVideo(v, data, "📘 Facebook Video")
}

// ── .ttdl / .ttvideo ─────────────────────────────────────────────────────────
// Usage: .ttdl <TikTok URL>

func ttdlHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .ttdl <TikTok URL>\nExample: .ttdl https://vm.tiktok.com/xxx")
	}

	ttURL := args[0]
	if !strings.Contains(ttURL, "tiktok.com") {
		return ReplyErr(v, "❌ Please provide a valid TikTok URL.")
	}

	Reply(v, "📥 Downloading TikTok video...")

	res, apiErr := apiGet("/api/download/tiktok", map[string]string{"url": ttURL})
	if apiErr == nil && res.bestURL() != "" {
		data, err := downloadBytesMedia(res.bestURL())
		if err == nil && len(data) > 0 {
			title := res.Title
			if title == "" {
				title = "TikTok Video"
			}
			return sendVideo(v, data, "🎵 *"+title+"*")
		}
	}

	Reply(v, "⏳ Trying alternate method...")
	data, err := ytdlpVideo(ttURL)
	if err != nil || len(data) == 0 {
		return ReplyErr(v, fmt.Sprintf("❌ TikTok download failed.\nAPI: %v\nyt-dlp: %v", apiErr, err))
	}
	return sendVideo(v, data, "🎵 TikTok Video")
}

// ── .igdl2 ───────────────────────────────────────────────────────────────────
// Usage: .igdl2 <Instagram URL>
// (named igdl2 to not conflict with existing igdl in instadl.go)

func igdl2Handler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .igdl2 <Instagram URL>\nExample: .igdl2 https://instagram.com/reel/xxx")
	}

	igURL := args[0]
	if !strings.Contains(igURL, "instagram.com") && !strings.Contains(igURL, "instagr.am") {
		return ReplyErr(v, "❌ Please provide a valid Instagram URL.")
	}

	Reply(v, "📥 Downloading Instagram media...")

	res, apiErr := apiGet("/api/download/instagram", map[string]string{"url": igURL})
	if apiErr == nil && res.bestURL() != "" {
		data, err := downloadBytesMedia(res.bestURL())
		if err == nil && len(data) > 0 {
			dlURL := res.bestURL()
			isVideo := strings.Contains(dlURL, ".mp4") ||
				strings.Contains(igURL, "/reel/") ||
				strings.Contains(igURL, "/tv/")
			if isVideo {
				return sendVideo(v, data, "📸 Instagram")
			}
			return sendImage(v, data, "📸 Instagram")
		}
	}

	Reply(v, "⏳ Trying alternate method...")
	data, err := ytdlpVideo(igURL)
	if err != nil || len(data) == 0 {
		return ReplyErr(v, fmt.Sprintf("❌ Instagram download failed.\nAPI: %v\nyt-dlp: %v", apiErr, err))
	}
	return sendVideo(v, data, "📸 Instagram")
}
