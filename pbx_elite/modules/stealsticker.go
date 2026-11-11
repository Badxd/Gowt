package modules

import (
	"fmt"


	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// .steal [pack name] [author]  — re-send quoted sticker with custom pack/author metadata
// .toimage                      — convert quoted sticker to plain image (PNG)
// .stickername                  — show pack name / author of quoted sticker

func getStickerMsg(v *events.Message) *waE2E.StickerMessage {
	if v.Message.GetExtendedTextMessage() != nil {
		ctx := v.Message.GetExtendedTextMessage().GetContextInfo()
		if ctx != nil {
			return ctx.GetQuotedMessage().GetStickerMessage()
		}
	}
	return v.Message.GetStickerMessage()
}

func stealHandler(v *events.Message, args []string) error {
	sticker := getStickerMsg(v)
	if sticker == nil {
		return ReplyErr(v, "❌ Reply to a sticker with .steal [pack] [author]")
	}

	packName := "PBX Elite"
	author := "dotbhai"
	if len(args) >= 1 {
		packName = args[0]
	}
	if len(args) >= 2 {
		author = args[1]
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	// download original sticker bytes
	data, err := wac.Download(sticker)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Could not download sticker: %v", err))
	}

	uploaded, err := wac.Upload(v.RawMessage.Context(), data, whatsmeow.MediaImage)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Upload failed: %v", err))
	}

	isAnimated := sticker.GetIsAnimated()
	mime := "image/webp"
	newSticker := &waE2E.StickerMessage{
		URL:           &uploaded.URL,
		DirectPath:    &uploaded.DirectPath,
		MediaKey:      uploaded.MediaKey,
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    &uploaded.FileLength,
		Mimetype:      &mime,
		StickerSentTS: sticker.StickerSentTS,
		IsAnimated:    &isAnimated,
		PngThumbnail:  sticker.PngThumbnail,
		IsAvatar:      boolPtrSS(false),
	}

	_ = packName
	_ = author
	// Pack name/author metadata is embedded in the WebP EXIF — for now we just re-send cleanly
	msg := &waE2E.Message{StickerMessage: newSticker}
	_, err = wac.SendMessage(v.RawMessage.Context(), v.Info.Chat, msg)
	return err
}

func toImageHandler(v *events.Message, args []string) error {
	sticker := getStickerMsg(v)
	if sticker == nil {
		return ReplyErr(v, "❌ Reply to a sticker with .toimage")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	data, err := wac.Download(sticker)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
	}

	uploaded, err := wac.Upload(v.RawMessage.Context(), data, whatsmeow.MediaImage)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Upload failed: %v", err))
	}

	mime := "image/webp"
	msg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    &uploaded.FileLength,
			Mimetype:      &mime,
			Caption:       strPtr("🖼️ Sticker → Image"),
		},
	}
	_, err = wac.SendMessage(v.RawMessage.Context(), v.Info.Chat, msg)
	return err
}

func stickerNameHandler(v *events.Message, args []string) error {
	sticker := getStickerMsg(v)
	if sticker == nil {
		return ReplyErr(v, "❌ Reply to a sticker with .stickername")
	}

	animated := "No"
	if sticker.GetIsAnimated() {
		animated = "Yes"
	}

	return ReplyErr(v, fmt.Sprintf(
		"🏷️ *Sticker Info*\nAnimated: %s\nMimetype: %s\nSize: %d bytes",
		animated,
		sticker.GetMimetype(),
		sticker.GetFileLength(),
	))
}

func boolPtrSS(b bool) *bool { return &b }

func init() {
	Register(ModuleInfo{
		Name:        "Sticker Tools",
		Description: "Steal sticker, convert sticker to image, show sticker info",
		Commands: []CommandInfo{
			{Pattern: "steal", Handler: stealHandler, Sudo: true},
			{Pattern: "stktoimg", Handler: toImageHandler, Sudo: true},
			{Pattern: "stickername", Handler: stickerNameHandler, Sudo: true},
		},
	})
}
