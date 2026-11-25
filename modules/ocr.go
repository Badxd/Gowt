package modules

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"PbxGb/client"

	"go.mau.fi/whatsmeow/types/events"
)

// .ocr — extract text from an image using api.ocr.space
// Set OCR_API_KEY env var on Heroku for a real key (get free at ocr.space).
// Falls back to the public demo key (rate-limited: ~500 req/month).

const ocrEndpoint = "https://api2.ocr.space/parse/image"

func getOCRKey() string {
	if k := os.Getenv("OCR_API_KEY"); k != "" {
		return k
	}
	return "helloworld" // ocr.space free demo key
}

type ocrResponse struct {
	ParsedResults []struct {
		ParsedText string `json:"ParsedText"`
		ErrorMessage string `json:"ErrorMessage"`
	} `json:"ParsedResults"`
	IsErroredOnProcessing bool   `json:"IsErroredOnProcessing"`
	ErrorMessage          string `json:"ErrorMessage"`
}

func runOCR(imgBytes []byte, mime string) (string, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	_ = w.WriteField("apikey", getOCRKey())
	_ = w.WriteField("language", "eng")
	_ = w.WriteField("isOverlayRequired", "false")

	ext := "jpg"
	if strings.Contains(mime, "png") {
		ext = "png"
	}

	fw, err := w.CreateFormFile("file", "image."+ext)
	if err != nil {
		return "", err
	}
	fw.Write(imgBytes)
	w.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ocrEndpoint, &b)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result ocrResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("OCR parse error: %s", err)
	}

	if result.IsErroredOnProcessing {
		return "", fmt.Errorf("OCR error: %s", result.ErrorMessage)
	}

	var texts []string
	for _, pr := range result.ParsedResults {
		if pr.ErrorMessage != "" {
			continue
		}
		t := strings.TrimSpace(pr.ParsedText)
		if t != "" {
			texts = append(texts, t)
		}
	}

	if len(texts) == 0 {
		return "", fmt.Errorf("no text found in image")
	}
	return strings.Join(texts, "\n"), nil
}

// .ocr — reply to or attach an image to extract text
func ocrHandler(v *events.Message, args []string) error {
	wac := client.Get()

	var imgBytes []byte
	var mime string
	var err error

	// Direct image attachment
	if img := v.Message.GetImageMessage(); img != nil {
		imgBytes, err = wac.Download(context.Background(), img)
		mime = img.GetMimetype()
		if err != nil {
			return ReplyErr(v, "❌ Failed to download image: "+err.Error())
		}
	}

	// Quoted image
	if imgBytes == nil {
		if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			if ctx := ext.GetContextInfo(); ctx != nil {
				if qm := ctx.GetQuotedMessage(); qm != nil {
					if qimg := qm.GetImageMessage(); qimg != nil {
						imgBytes, err = wac.Download(context.Background(), qimg)
						mime = qimg.GetMimetype()
						if err != nil {
							return ReplyErr(v, "❌ Failed to download quoted image: "+err.Error())
						}
					}
				}
			}
		}
	}

	if len(imgBytes) == 0 {
		return ReplyErr(v, "Usage: Reply to an image with .ocr\nExtracts text from the image.")
	}

	if mime == "" {
		mime = "image/jpeg"
	}

	SendText(v, "🔍 Extracting text from image...")

	text, err := runOCR(imgBytes, mime)
	if err != nil {
		return ReplyErr(v, "❌ OCR failed: "+err.Error())
	}

	return ReplyErr(v, "📝 *Extracted Text:*\n\n"+text)
}

func init() {
	Register(ModuleInfo{
		Name:        "OCR",
		Description: "Extract text from images using OCR",
		Commands: []CommandInfo{
			{Pattern: "ocr", Handler: ocrHandler, Sudo: true},
		},
	})
}
