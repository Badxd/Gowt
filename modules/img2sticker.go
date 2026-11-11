package modules

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"os"
	"os/exec"
	"time"


	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// .sticker / .s  — convert replied/attached image to WhatsApp sticker (512x512 WebP)

// imgToWebP: decode any image, crop to square, scale to 512x512, encode as WebP via cwebp.
func imgToWebP(imgData []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	size := w
	if h < size {
		size = h
	}

	// centre crop
	x0 := b.Min.X + (w-size)/2
	y0 := b.Min.Y + (h-size)/2

	cropped := image.NewNRGBA(image.Rect(0, 0, 512, 512))
	for dy := 0; dy < 512; dy++ {
		for dx := 0; dx < 512; dx++ {
			sx := x0 + (dx*size)/512
			sy := y0 + (dy*size)/512
			cropped.Set(dx, dy, img.At(sx, sy))
		}
	}

	// write PNG to temp file (cwebp reads from file)
	tmp, err := os.CreateTemp("", "sticker-*.png")
	if err != nil {
		return nil, fmt.Errorf("tmp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	if err := png.Encode(tmp, cropped); err != nil {
		tmp.Close()
		return nil, fmt.Errorf("png encode: %w", err)
	}
	tmp.Close()

	// encode WebP via cwebp
	out, err := exec.Command("cwebp", "-lossless", "-quiet", tmp.Name(), "-o", "-").Output()
	if err != nil {
		// fallback: ffmpeg
		out, err = exec.Command(
			"ffmpeg", "-y", "-i", tmp.Name(),
			"-vf", "scale=512:512",
			"-f", "webp", "pipe:1",
		).Output()
		if err != nil {
			return nil, fmt.Errorf("webp encode failed (need cwebp or ffmpeg): %w", err)
		}
	}

	return out, nil
}

func imgStickerHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	var imgData []byte

	// direct image
	if img := v.Message.GetImageMessage(); img != nil {
		data, err := wac.Download(ctx, img)
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
		}
		imgData = data
	}

	// quoted image
	if imgData == nil {
		if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			if ci := ext.GetContextInfo(); ci != nil {
				if qm := ci.GetQuotedMessage(); qm != nil {
					if img := qm.GetImageMessage(); img != nil {
						data, err := wac.Download(ctx, img)
						if err != nil {
							return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
						}
						imgData = data
					}
				}
			}
		}
	}

	if imgData == nil {
		return ReplyErr(v, "❌ Reply to an image to convert it to a sticker.")
	}

	webpData, err := imgToWebP(imgData)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Conversion failed: %v\n_Requires `cwebp` or `ffmpeg` installed._", err))
	}

	uploaded, err := wac.Upload(ctx, webpData, whatsmeow.MediaImage)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Upload failed: %v", err))
	}

	animated := false
	mime := "image/webp"
	fl := uploaded.FileLength

	msg := &waE2E.Message{
		StickerMessage: &waE2E.StickerMessage{
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    &fl,
			Mimetype:      &mime,
			IsAnimated:    &animated,
		},
	}

	_, err = wac.SendMessage(ctx, v.Info.Chat, msg)
	return err
}

func init() {
	Register(ModuleInfo{
		Name:        "Image to Sticker",
		Description: "Convert images to WhatsApp stickers (512×512 WebP)",
		Commands: []CommandInfo{
			{Pattern: "imgsticker", Handler: imgStickerHandler, Sudo: true},
			{Pattern: "imgs",       Handler: imgStickerHandler, Sudo: true},
		},
	})
}
