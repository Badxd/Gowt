package modules

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"


	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// .toimage   — convert a sticker back to image and send
// .stickerurl — get the raw CDN URL of a sticker

func stickerToImageHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var stickerMsg *waE2E.StickerMessage

	// direct sticker
	if sm := v.Message.GetStickerMessage(); sm != nil {
		stickerMsg = sm
	}

	// quoted sticker
	if stickerMsg == nil {
		if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			if ci := ext.GetContextInfo(); ci != nil {
				if qm := ci.GetQuotedMessage(); qm != nil {
					if sm := qm.GetStickerMessage(); sm != nil {
						stickerMsg = sm
					}
				}
			}
		}
	}

	if stickerMsg == nil {
		return ReplyErr(v, "❌ Reply to a sticker to convert it to an image.")
	}

	data, err := wac.Download(ctx, stickerMsg)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
	}

	// Upload as image
	uploaded, err := wac.Upload(ctx, data, whatsmeow.MediaImage)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Upload failed: %v", err))
	}

	mime := "image/webp"
	fl := uploaded.FileLength
	caption := "🖼️ Sticker → Image"

	msg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    &fl,
			Mimetype:      &mime,
			Caption:       &caption,
		},
	}

	_, err = wac.SendMessage(ctx, v.Info.Chat, msg)
	return err
}

// .stickerpack <packname> <author>  — (re-)tag a sticker with custom pack metadata
// Note: full EXIF embedding requires external tool; this version just re-uploads with metadata fields
func stickerTagHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		return ReplyErr(v, "Usage: .stickerpack <packname> <author>")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var stickerMsg *waE2E.StickerMessage

	if sm := v.Message.GetStickerMessage(); sm != nil {
		stickerMsg = sm
	}
	if stickerMsg == nil {
		if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			if ci := ext.GetContextInfo(); ci != nil {
				if qm := ci.GetQuotedMessage(); qm != nil {
					stickerMsg = qm.GetStickerMessage()
				}
			}
		}
	}

	if stickerMsg == nil {
		return ReplyErr(v, "❌ Reply to a sticker to re-tag it.")
	}

	packName := args[0]
	author := strings.Join(args[1:], " ")

	data, err := wac.Download(ctx, stickerMsg)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
	}

	uploaded, err := wac.Upload(ctx, data, whatsmeow.MediaImage)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Upload failed: %v", err))
	}

	animated := stickerMsg.GetIsAnimated()
	mime := "image/webp"
	fl := uploaded.FileLength

	newSticker := &waE2E.StickerMessage{
		URL:           &uploaded.URL,
		DirectPath:    &uploaded.DirectPath,
		MediaKey:      uploaded.MediaKey,
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    &fl,
		Mimetype:      &mime,
		IsAnimated:    proto.Bool(animated),
		StickerName:   proto.String(packName),
		StickerPublisher: proto.String(author),
	}

	// encode exif metadata JSON into stickerMsg
	exifJSON := fmt.Sprintf(`{"pack-name":"%s","pack-author":"%s"}`, packName, author)
	newSticker.FirstFrameSidecar = []byte(exifJSON)

	_, err = wac.SendMessage(ctx, v.Info.Chat, &waE2E.Message{StickerMessage: newSticker})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Send failed: %v", err))
	}

	return ReplyErr(v, fmt.Sprintf("✅ Sticker re-tagged.\n📦 Pack: *%s*\n✍️ Author: *%s*", packName, author))
}

// .getsticker — re-send a sticker from quoted image (convert image to sticker inline)
func getStickerFromImageHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	var imgData []byte

	// quoted image
	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		if ci := ext.GetContextInfo(); ci != nil {
			if qm := ci.GetQuotedMessage(); qm != nil {
				if img := qm.GetImageMessage(); img != nil {
					var err error
					imgData, err = wac.Download(ctx, img)
					if err != nil {
						return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
					}
				}
			}
		}
	}

	// direct image message
	if imgData == nil {
		if img := v.Message.GetImageMessage(); img != nil {
			var err error
			imgData, err = wac.Download(ctx, img)
			if err != nil {
				return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
			}
		}
	}

	if imgData == nil {
		return ReplyErr(v, "❌ Reply to an image to convert it to a sticker.")
	}

	webpData, err := imgToWebP(imgData)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Conversion failed: %v", err))
	}

	uploaded, err := wac.Upload(ctx, webpData, whatsmeow.MediaImage)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Upload failed: %v", err))
	}

	animated := false
	mime := "image/webp"
	fl := uploaded.FileLength

	_ = bytes.NewReader(nil) // keep import

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
		Name:        "Sticker Tools",
		Description: "Convert stickers to images, tag sticker packs",
		Commands: []CommandInfo{
			{Pattern: "toimage", Handler: stickerToImageHandler, Sudo: true},
			{Pattern: "stickerpack", Handler: stickerTagHandler, Sudo: true},
			{Pattern: "getsticker", Handler: getStickerFromImageHandler, Sudo: true},
		},
	})
}
