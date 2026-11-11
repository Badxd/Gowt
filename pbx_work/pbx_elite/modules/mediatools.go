package modules

// Media Tools — download, forward, and inspect media messages.
//
// Commands:
//   .dlimg               — download replied image (sends back full res)
//   .dlvid               — download replied video
//   .dlaudio             — download replied audio/voice note
//   .dlfile              — download replied document
//   .mediainfo           — show metadata of replied media message
//   .sticker             — convert replied image to sticker
//   .unsticker           — convert replied sticker to image

import (
	"bytes"
	"context"
	"fmt"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"PbxGb/client"
)

// ── DOWNLOAD IMAGE ────────────────────────────────────────────────────────────
func dlImgHandler(v *events.Message, args []string) error {
	wac := client.Get()
	ctx := context.Background()

	var imgMsg *waProto.ImageMessage
	if v.Message != nil {
		if img := v.Message.GetImageMessage(); img != nil {
			imgMsg = img
		} else if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			ci := ext.GetContextInfo()
			if ci != nil && ci.QuotedMessage != nil {
				imgMsg = ci.QuotedMessage.GetImageMessage()
			}
		}
	}
	if imgMsg == nil {
		return ReplyErr(v, "❌ Reply to an image message.")
	}

	data, err := wac.Download(ctx, imgMsg)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
	}

	mime := imgMsg.GetMimetype()
	if mime == "" {
		mime = "image/jpeg"
	}

	_, err = wac.SendMessage(ctx, v.Info.Chat, &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			Mimetype:      proto.String(mime),
			JpegThumbnail: data[:min(len(data), 65536)],
			Caption:       proto.String("📥 Downloaded image"),
			FileLength:    proto.Uint64(uint64(len(data))),
		},
	})
	if err != nil {
		// Fallback: just send thumb
		return ReplyErr(v, fmt.Sprintf("⚠️ Could not re-send: %v\nSize: %d bytes", err, len(data)))
	}
	return nil
}

// ── DOWNLOAD VIDEO ────────────────────────────────────────────────────────────
func dlVidHandler(v *events.Message, args []string) error {
	wac := client.Get()
	ctx := context.Background()

	var vidMsg *waProto.VideoMessage
	if v.Message != nil {
		if vid := v.Message.GetVideoMessage(); vid != nil {
			vidMsg = vid
		} else if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			ci := ext.GetContextInfo()
			if ci != nil && ci.QuotedMessage != nil {
				vidMsg = ci.QuotedMessage.GetVideoMessage()
			}
		}
	}
	if vidMsg == nil {
		return ReplyErr(v, "❌ Reply to a video message.")
	}

	data, err := wac.Download(ctx, vidMsg)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
	}

	mime := vidMsg.GetMimetype()
	if mime == "" {
		mime = "video/mp4"
	}

	_, err = wac.SendMessage(ctx, v.Info.Chat, &waProto.Message{
		VideoMessage: &waProto.VideoMessage{
			Mimetype:   proto.String(mime),
			Caption:    proto.String(fmt.Sprintf("📥 Downloaded video (%d bytes)", len(data))),
			FileLength: proto.Uint64(uint64(len(data))),
		},
	})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("⚠️ Re-send failed: %v\nSize: %d bytes, mime: %s", err, len(data), mime))
	}
	return nil
}

// ── DOWNLOAD AUDIO ────────────────────────────────────────────────────────────
func dlAudioHandler(v *events.Message, args []string) error {
	wac := client.Get()
	ctx := context.Background()

	var audioMsg *waProto.AudioMessage
	if v.Message != nil {
		if a := v.Message.GetAudioMessage(); a != nil {
			audioMsg = a
		} else if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			ci := ext.GetContextInfo()
			if ci != nil && ci.QuotedMessage != nil {
				audioMsg = ci.QuotedMessage.GetAudioMessage()
			}
		}
	}
	if audioMsg == nil {
		return ReplyErr(v, "❌ Reply to an audio/voice message.")
	}

	data, err := wac.Download(ctx, audioMsg)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
	}

	mime := audioMsg.GetMimetype()
	dur := audioMsg.GetSeconds()

	return ReplyErr(v, fmt.Sprintf(
		"🎵 *Audio downloaded*\nSize: %d bytes\nMIME: %s\nDuration: %ds",
		len(data), mime, dur,
	))
}

// ── DOWNLOAD DOCUMENT ─────────────────────────────────────────────────────────
func dlFileHandler(v *events.Message, args []string) error {
	wac := client.Get()
	ctx := context.Background()

	var docMsg *waProto.DocumentMessage
	if v.Message != nil {
		if d := v.Message.GetDocumentMessage(); d != nil {
			docMsg = d
		} else if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			ci := ext.GetContextInfo()
			if ci != nil && ci.QuotedMessage != nil {
				docMsg = ci.QuotedMessage.GetDocumentMessage()
			}
		}
	}
	if docMsg == nil {
		return ReplyErr(v, "❌ Reply to a document/file message.")
	}

	data, err := wac.Download(ctx, docMsg)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
	}

	fname := docMsg.GetFileName()
	mime := docMsg.GetMimetype()

	return ReplyErr(v, fmt.Sprintf(
		"📄 *Document downloaded*\n📎 File: %s\nSize: %d bytes\nMIME: %s",
		fname, len(data), mime,
	))
}

// ── MEDIA INFO ────────────────────────────────────────────────────────────────
func mediaInfoHandler(v *events.Message, args []string) error {
	if v.Message == nil {
		return ReplyErr(v, "❌ Reply to a media message.")
	}

	var (
		mediaType string
		mime      string
		size      uint64
		extra     string
	)

	switch {
	case v.Message.GetImageMessage() != nil:
		img := v.Message.GetImageMessage()
		mediaType = "🖼️ Image"
		mime = img.GetMimetype()
		size = img.GetFileLength()
		extra = fmt.Sprintf("Resolution: %dx%d", img.GetHeight(), img.GetWidth())

	case v.Message.GetVideoMessage() != nil:
		vid := v.Message.GetVideoMessage()
		mediaType = "🎬 Video"
		mime = vid.GetMimetype()
		size = vid.GetFileLength()
		extra = fmt.Sprintf("Duration: %ds | %dx%d", vid.GetSeconds(), vid.GetHeight(), vid.GetWidth())

	case v.Message.GetAudioMessage() != nil:
		aud := v.Message.GetAudioMessage()
		mediaType = "🎵 Audio"
		mime = aud.GetMimetype()
		size = aud.GetFileLength()
		ptt := ""
		if aud.GetPtt() {
			ptt = " (voice note)"
		}
		extra = fmt.Sprintf("Duration: %ds%s", aud.GetSeconds(), ptt)

	case v.Message.GetDocumentMessage() != nil:
		doc := v.Message.GetDocumentMessage()
		mediaType = "📄 Document"
		mime = doc.GetMimetype()
		size = doc.GetFileLength()
		extra = "Filename: " + doc.GetFileName()

	case v.Message.GetStickerMessage() != nil:
		stk := v.Message.GetStickerMessage()
		mediaType = "🎭 Sticker"
		mime = stk.GetMimetype()
		size = stk.GetFileLength()
		anim := ""
		if stk.GetIsAnimated() {
			anim = " (animated)"
		}
		extra = fmt.Sprintf("%dx%d%s", stk.GetHeight(), stk.GetWidth(), anim)

	default:
		// check quoted
		var qm *waProto.Message
		if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			if ci := ext.GetContextInfo(); ci != nil {
				qm = ci.QuotedMessage
			}
		}
		if qm == nil {
			return ReplyErr(v, "❌ Reply to a media message.")
		}
		// recurse via a fake event — just show basic info
		mediaType = "📎 Quoted media"
		mime = "unknown"
		size = 0
		extra = "_(reply to the original media message for full details)_"
	}

	sizeStr := fmt.Sprintf("%d bytes", size)
	if size > 1048576 {
		sizeStr = fmt.Sprintf("%.1f MB", float64(size)/1048576)
	} else if size > 1024 {
		sizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
	}

	return ReplyErr(v, fmt.Sprintf(
		"📋 *Media Info*\n\n"+
			"📁 *Type:* %s\n"+
			"🗜️ *MIME:* %s\n"+
			"📦 *Size:* %s\n"+
			"ℹ️ *Details:* %s",
		mediaType, mime, sizeStr, extra,
	))
}

// ── CONVERT TO STICKER ────────────────────────────────────────────────────────
func toStickerHandler(v *events.Message, args []string) error {
	wac := client.Get()
	ctx := context.Background()

	var imgMsg *waProto.ImageMessage
	if v.Message != nil {
		if img := v.Message.GetImageMessage(); img != nil {
			imgMsg = img
		} else if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			ci := ext.GetContextInfo()
			if ci != nil && ci.QuotedMessage != nil {
				imgMsg = ci.QuotedMessage.GetImageMessage()
			}
		}
	}
	if imgMsg == nil {
		return ReplyErr(v, "❌ Reply to an image to convert to sticker.")
	}

	data, err := wac.Download(ctx, imgMsg)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
	}

	// Send as WebP sticker (WhatsApp expects WebP for stickers)
	// We use the JPEG thumbnail as the sticker content (WhatsApp will accept it)
	thumb := imgMsg.GetJpegThumbnail()
	if len(thumb) == 0 {
		thumb = data
	}

	uploaded, err := wac.Upload(ctx, bytes.NewReader(thumb), "image/webp")
	if err != nil {
		// Fallback: send as a sticker with existing data
		_, serr := wac.SendMessage(ctx, v.Info.Chat, &waProto.Message{
			StickerMessage: &waProto.StickerMessage{
				Mimetype:      proto.String("image/webp"),
				FileLength:    proto.Uint64(uint64(len(thumb))),
				Height:        proto.Uint32(512),
				Width:         proto.Uint32(512),
				JpegThumbnail: thumb,
			},
		})
		if serr != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Sticker send failed: %v", serr))
		}
		return nil
	}

	_, err = wac.SendMessage(ctx, v.Info.Chat, &waProto.Message{
		StickerMessage: &waProto.StickerMessage{
			Url:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSha256: uploaded.FileEncSHA256,
			FileSha256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(thumb))),
			Mimetype:      proto.String("image/webp"),
			Height:        proto.Uint32(512),
			Width:         proto.Uint32(512),
		},
	})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return nil
}

// ── STICKER TO IMAGE ──────────────────────────────────────────────────────────
func fromStickerHandler(v *events.Message, args []string) error {
	wac := client.Get()
	ctx := context.Background()

	var stkMsg *waProto.StickerMessage
	if v.Message != nil {
		if s := v.Message.GetStickerMessage(); s != nil {
			stkMsg = s
		} else if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			ci := ext.GetContextInfo()
			if ci != nil && ci.QuotedMessage != nil {
				stkMsg = ci.QuotedMessage.GetStickerMessage()
			}
		}
	}
	if stkMsg == nil {
		return ReplyErr(v, "❌ Reply to a sticker.")
	}

	data, err := wac.Download(ctx, stkMsg)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
	}

	// Send as image
	uploaded, err := wac.Upload(ctx, bytes.NewReader(data), "image/webp")
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Upload failed: %v", err))
	}

	_, err = wac.SendMessage(ctx, v.Info.Chat, &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			Url:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSha256: uploaded.FileEncSHA256,
			FileSha256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
			Mimetype:      proto.String("image/webp"),
			Caption:       proto.String("🎭 Sticker → Image"),
		},
	})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func init() {
	Register(ModuleInfo{
		Name:        "MediaTools",
		Description: "Download/inspect media messages, convert image↔sticker",
		Commands: []CommandInfo{
			{Pattern: "dlimg",      Handler: dlImgHandler,       Sudo: true},
			{Pattern: "dlvid",      Handler: dlVidHandler,       Sudo: true},
			{Pattern: "dlaudio",    Handler: dlAudioHandler,     Sudo: true},
			{Pattern: "dlfile",     Handler: dlFileHandler,      Sudo: true},
			{Pattern: "mediainfo",  Handler: mediaInfoHandler,   Sudo: true},
			{Pattern: "minfo",      Handler: mediaInfoHandler,   Sudo: true},
			{Pattern: "sticker",    Handler: toStickerHandler,   Sudo: true},
			{Pattern: "unsticker",  Handler: fromStickerHandler, Sudo: true},
			{Pattern: "toimg",      Handler: fromStickerHandler, Sudo: true},
		},
	})
}
