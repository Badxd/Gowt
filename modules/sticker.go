package modules

// PBX Elite — Sticker Module (Heroku-safe, pure-Go, no ffmpeg/cwebp)
//
// .sticker / .s   — reply to an image → convert to sticker (pure Go)
// Animated sticker from video is NOT supported on Heroku (needs ffmpeg).

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func stickerHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	quoted := GetQuotedMessage(v)

	if quoted == nil {
		// direct image in this message
		if v.Message.GetImageMessage() != nil {
			return handleStickerFromMsg(v, wac)
		}
		return ReplyErr(v,
			"❌ Reply to an image to make a sticker.\n"+
				"Example: reply to any image + `.sticker`")
	}

	// IMAGE → STATIC STICKER
	if img := quoted.GetImageMessage(); img != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		imgData, err := wac.Download(ctx, img)
		if err != nil {
			return ReplyErr(v, "❌ Could not download image: "+err.Error())
		}
		return convertAndSendStickerCtx(ctx, v, wac, imgData)
	}

	// VIDEO → not supported on Heroku
	if quoted.GetVideoMessage() != nil {
		return ReplyErr(v,
			"⚠️ *Animated stickers from video need ffmpeg.*\n"+
				"ffmpeg is not available on Heroku.\n\n"+
				"💡 To add ffmpeg to Heroku:\n"+
				"```\nheroku buildpacks:add https://github.com/jonathanong/heroku-buildpack-ffmpeg-latest\n```\n"+
				"Then redeploy the bot.")
	}

	// STICKER → re-send as sticker
	if stk := quoted.GetStickerMessage(); stk != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		stickerData, err := wac.Download(ctx, stk)
		if err != nil {
			return ReplyErr(v, "❌ Could not download sticker.")
		}
		return convertAndSendStickerCtx(ctx, v, wac, stickerData)
	}

	return ReplyErr(v, "❌ Reply to an image to create a sticker.")
}

func handleStickerFromMsg(v *events.Message, wac *whatsmeow.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	imgData, err := wac.Download(ctx, v.Message.GetImageMessage())
	if err != nil {
		return ReplyErr(v, "❌ Download failed: "+err.Error())
	}
	return convertAndSendStickerCtx(ctx, v, wac, imgData)
}

func init() {
	Register(ModuleInfo{
		Name:        "Sticker",
		Description: "Convert image to WhatsApp sticker (.sticker / .s) — pure Go",
		Commands: []CommandInfo{
			{Pattern: "sticker", Handler: stickerHandler, Sudo: true},
			{Pattern: "s", Handler: stickerHandler, Sudo: true},
		},
	})
}
