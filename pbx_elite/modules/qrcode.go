package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"PbxGb/client"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

const qrGenerateAPI = "https://api.qrserver.com/v1/create-qr-code/"

func generateQRCode(text string, size int) ([]byte, error) {
	params := url.Values{}
	params.Set("size", fmt.Sprintf("%dx%d", size, size))
	params.Set("data", text)
	params.Set("format", "png")
	params.Set("margin", "10")
	endpoint := qrGenerateAPI + "?" + params.Encode()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("QR API returned %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func qrHandler(v *events.Message, args []string) error {
	wac := client.Get()

	var text string
	if len(args) > 0 {
		text = strings.Join(args, " ")
	} else if q := getQuotedText(v); q != "" {
		text = q
	}
	if text == "" {
		return ReplyErr(v, "Usage: .qr <text or URL>")
	}

	imgBytes, err := generateQRCode(text, 400)
	if err != nil {
		return ReplyErr(v, "❌ Failed to generate QR: "+err.Error())
	}

	uploaded, err := wac.Upload(context.Background(), imgBytes, whatsmeow.MediaImage)
	if err != nil {
		return ReplyErr(v, "❌ Upload failed: "+err.Error())
	}

	capped := text
	if len(capped) > 50 {
		capped = capped[:50] + "..."
	}
	caption := "📱 *QR Code:* " + capped
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

// decode QR from image bytes via goqr.me
func decodeQRBytes(imgBytes []byte) (string, error) {
	var buf strings.Builder
	// goqr.me requires multipart form
	boundary := "----QRBoundary"
	body := "--" + boundary + "\r\n"
	body += "Content-Disposition: form-data; name=\"file\"; filename=\"qr.png\"\r\n"
	body += "Content-Type: image/png\r\n\r\n"

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_ = buf
	pr, pw := io.Pipe()
	go func() {
		pw.Write([]byte(body))
		pw.Write(imgBytes)
		pw.Write([]byte("\r\n--" + boundary + "--\r\n"))
		pw.Close()
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.qrserver.com/v1/read-qr-code/", pr)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)

	type qrSym struct {
		Data  string `json:"data"`
		Error string `json:"error"`
	}
	type qrEntry struct {
		Symbol []qrSym `json:"symbol"`
	}
	var results []qrEntry
	if err := json.Unmarshal(data, &results); err != nil || len(results) == 0 {
		return "", fmt.Errorf("no QR data found")
	}
	if len(results[0].Symbol) == 0 || results[0].Symbol[0].Data == "" {
		return "", fmt.Errorf("could not decode QR")
	}
	if results[0].Symbol[0].Error != "" {
		return "", fmt.Errorf(results[0].Symbol[0].Error)
	}
	return results[0].Symbol[0].Data, nil
}

func qrDecodeHandler(v *events.Message, args []string) error {
	wac := client.Get()

	// direct image attachment
	if img := v.Message.GetImageMessage(); img != nil {
		imgBytes, err := wac.Download(img)
		if err != nil {
			return ReplyErr(v, "❌ Download failed: "+err.Error())
		}
		text, err := decodeQRBytes(imgBytes)
		if err != nil {
			return ReplyErr(v, "❌ Could not decode QR: "+err.Error())
		}
		return ReplyErr(v, "🔍 *QR Decoded:*\n"+text)
	}

	// quoted image
	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		if ci := ext.GetContextInfo(); ci != nil {
			if qm := ci.GetQuotedMessage(); qm != nil {
				if qimg := qm.GetImageMessage(); qimg != nil {
					imgBytes, err := wac.Download(qimg)
					if err == nil {
						text, err2 := decodeQRBytes(imgBytes)
						if err2 == nil {
							return ReplyErr(v, "🔍 *QR Decoded:*\n"+text)
						}
					}
				}
			}
		}
	}

	return ReplyErr(v, "Reply to a QR code image with .qrdecode")
}

func init() {
	Register(ModuleInfo{
		Name:        "QRCode",
		Description: "Generate and decode QR codes",
		Commands: []CommandInfo{
			{Pattern: "qr", Handler: qrHandler, Sudo: true},
			{Pattern: "qrdecode", Handler: qrDecodeHandler, Sudo: true},
		},
	})
}
