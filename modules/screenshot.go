package modules

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"PbxGb/client"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// .ss <url> — take a screenshot of a website using screenshotmachine.com or similar free API

const ssAPI = "https://mini.s-shot.ru/1024x768/PNG/1024/Z100/?"

// fetchScreenshot fetches a PNG screenshot of the given URL
func fetchScreenshot(targetURL string) ([]byte, error) {
	apiURL := ssAPI + targetURL

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 PBX-Elite/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("screenshot service returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(data) < 1000 {
		return nil, fmt.Errorf("screenshot too small — site may be unreachable")
	}
	return data, nil
}

// .ss <url>
func screenshotHandler(v *events.Message, args []string) error {
	wac := client.Get()

	if len(args) == 0 {
		return ReplyErr(v, "Usage: .ss <url>\nExample: .ss https://example.com")
	}

	targetURL := args[0]
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	SendText(v, "📸 Taking screenshot of "+targetURL+"...")

	imgBytes, err := fetchScreenshot(targetURL)
	if err != nil {
		return ReplyErr(v, "❌ Screenshot failed: "+err.Error())
	}

	uploaded, err := wac.Upload(context.Background(), imgBytes, whatsmeow.MediaImage)
	if err != nil {
		return ReplyErr(v, "❌ Upload failed: "+err.Error())
	}

	caption := fmt.Sprintf("🌐 Screenshot of %s", targetURL)
	imgMsg := &waE2E.ImageMessage{
		URL:           &uploaded.URL,
		DirectPath:    &uploaded.DirectPath,
		MediaKey:      uploaded.MediaKey,
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    &uploaded.FileLength,
		Mimetype:      strPtr("image/png"),
		Caption:       &caption,
	}

	_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waE2E.Message{ImageMessage: imgMsg})
	return err
}

var _ = bytes.NewReader // suppress unused

func init() {
	Register(ModuleInfo{
		Name:        "Screenshot",
		Description: "Take a screenshot of any website",
		Commands: []CommandInfo{
			{Pattern: "ss", Handler: screenshotHandler, Sudo: true},
		},
	})
}
