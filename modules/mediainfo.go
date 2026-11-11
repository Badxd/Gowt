package modules

import (
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

// .mediainfo — show metadata of a quoted/attached media message

func mediaInfoHandler(v *events.Message, args []string) error {
	// check quoted first, then direct message
	msg := v.Message
	if v.Message.GetExtendedTextMessage() != nil {
		ctx := v.Message.GetExtendedTextMessage().GetContextInfo()
		if ctx != nil && ctx.GetQuotedMessage() != nil {
			msg = ctx.GetQuotedMessage()
		}
	}

	var lines []string

	if img := msg.GetImageMessage(); img != nil {
		lines = append(lines,
			"📷 *Type:* Image",
			fmt.Sprintf("📦 *Mime:* %s", img.GetMimetype()),
			fmt.Sprintf("📐 *Dimensions:* %dx%d", img.GetWidth(), img.GetHeight()),
			fmt.Sprintf("💾 *Size:* %s", formatBytes(img.GetFileLength())),
			fmt.Sprintf("📝 *Caption:* %s", nonEmpty(img.GetCaption())),
		)
	} else if vid := msg.GetVideoMessage(); vid != nil {
		lines = append(lines,
			"🎬 *Type:* Video",
			fmt.Sprintf("📦 *Mime:* %s", vid.GetMimetype()),
			fmt.Sprintf("📐 *Dimensions:* %dx%d", vid.GetWidth(), vid.GetHeight()),
			fmt.Sprintf("⏱️ *Duration:* %ds", vid.GetSeconds()),
			fmt.Sprintf("💾 *Size:* %s", formatBytes(vid.GetFileLength())),
			fmt.Sprintf("📝 *Caption:* %s", nonEmpty(vid.GetCaption())),
		)
	} else if aud := msg.GetAudioMessage(); aud != nil {
		audType := "Audio"
		if aud.GetPTT() {
			audType = "Voice Note (PTT)"
		}
		lines = append(lines,
			fmt.Sprintf("🎵 *Type:* %s", audType),
			fmt.Sprintf("📦 *Mime:* %s", aud.GetMimetype()),
			fmt.Sprintf("⏱️ *Duration:* %ds", aud.GetSeconds()),
			fmt.Sprintf("💾 *Size:* %s", formatBytes(aud.GetFileLength())),
		)
	} else if doc := msg.GetDocumentMessage(); doc != nil {
		lines = append(lines,
			"📄 *Type:* Document",
			fmt.Sprintf("📦 *Mime:* %s", doc.GetMimetype()),
			fmt.Sprintf("📁 *Filename:* %s", nonEmpty(doc.GetFileName())),
			fmt.Sprintf("💾 *Size:* %s", formatBytes(doc.GetFileLength())),
			fmt.Sprintf("📝 *Caption:* %s", nonEmpty(doc.GetCaption())),
		)
	} else if stk := msg.GetStickerMessage(); stk != nil {
		animated := "No"
		if stk.GetIsAnimated() {
			animated = "Yes"
		}
		lines = append(lines,
			"🎴 *Type:* Sticker",
			fmt.Sprintf("📦 *Mime:* %s", stk.GetMimetype()),
			fmt.Sprintf("🎞️ *Animated:* %s", animated),
			fmt.Sprintf("💾 *Size:* %s", formatBytes(stk.GetFileLength())),
		)
	} else if txt := msg.GetConversation(); txt != "" {
		lines = append(lines,
			"💬 *Type:* Text",
			fmt.Sprintf("📝 *Content:* %s", truncate(txt, 80)),
			fmt.Sprintf("🔢 *Length:* %d chars", len(txt)),
		)
	} else {
		return ReplyErr(v, "❌ Reply to a media message to get its info.")
	}

	return ReplyErr(v, strings.Join(lines, "\n"))
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func nonEmpty(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func init() {
	Register(ModuleInfo{
		Name:        "Media Info",
		Description: "Show metadata of a quoted media message",
		Commands: []CommandInfo{
			{Pattern: "mediainfo", Handler: mediaInfoHandler, Sudo: true},
		},
	})
}
