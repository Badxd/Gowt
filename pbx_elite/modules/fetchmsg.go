package modules

import (
	"context"
	"fmt"
	"strings"
	"time"


	"go.mau.fi/whatsmeow/types/events"
)

// .quoted   — show full content of the replied-to message (useful for long/hidden text)
// .msginfo  — show metadata of the replied-to message

func quotedHandler(v *events.Message, args []string) error {
	ext := v.Message.GetExtendedTextMessage()
	if ext == nil {
		return ReplyErr(v, "❌ Reply to a message to view its content.")
	}

	ci := ext.GetContextInfo()
	if ci == nil {
		return ReplyErr(v, "❌ No context info in this reply.")
	}

	qm := ci.GetQuotedMessage()
	if qm == nil {
		return ReplyErr(v, "❌ No quoted message found.")
	}

	// extract text from various message types
	text := ""
	msgType := "unknown"

	if t := qm.GetConversation(); t != "" {
		text = t
		msgType = "text"
	} else if ext2 := qm.GetExtendedTextMessage(); ext2 != nil {
		text = ext2.GetText()
		msgType = "text"
	} else if img := qm.GetImageMessage(); img != nil {
		text = img.GetCaption()
		msgType = "image"
	} else if vid := qm.GetVideoMessage(); vid != nil {
		text = vid.GetCaption()
		msgType = "video"
	} else if doc := qm.GetDocumentMessage(); doc != nil {
		text = doc.GetCaption()
		msgType = "document"
	} else if audio := qm.GetAudioMessage(); audio != nil {
		msgType = "audio"
	} else if sticker := qm.GetStickerMessage(); sticker != nil {
		msgType = "sticker"
	}

	sender := ""
	if ci.Participant != nil {
		jidParts := strings.SplitN(*ci.Participant, "@", 2)
		sender = "+" + jidParts[0]
	}

	lines := []string{fmt.Sprintf("📨 *Quoted Message (%s)*", msgType)}
	if sender != "" {
		lines = append(lines, fmt.Sprintf("👤 From: %s", sender))
	}
	if text != "" {
		lines = append(lines, "", text)
	} else {
		lines = append(lines, "_[no text content]_")
	}

	return ReplyErr(v, strings.Join(lines, "\n"))
}

func msgInfoHandler(v *events.Message, args []string) error {
	ext := v.Message.GetExtendedTextMessage()
	if ext == nil {
		// show info of the current message
		return showMsgInfo(v, v.Info.ID.String(), v.Info.Sender.String(), v.Info.Timestamp)
	}

	ci := ext.GetContextInfo()
	if ci == nil || ci.StanzaID == nil {
		return showMsgInfo(v, v.Info.ID.String(), v.Info.Sender.String(), v.Info.Timestamp)
	}

	sender := ""
	if ci.Participant != nil {
		sender = *ci.Participant
	}

	return showMsgInfo(v, *ci.StanzaID, sender, time.Time{})
}

func showMsgInfo(v *events.Message, id, sender string, ts time.Time) error {
	lines := []string{"ℹ️ *Message Info*", ""}
	lines = append(lines, fmt.Sprintf("🆔 ID: `%s`", id))
	if sender != "" {
		parts := strings.SplitN(sender, "@", 2)
		lines = append(lines, fmt.Sprintf("👤 Sender: +%s", parts[0]))
	}
	if !ts.IsZero() {
		lines = append(lines, fmt.Sprintf("🕒 Time: %s", ts.Format("02 Jan 2006, 15:04:05")))
	}
	lines = append(lines, fmt.Sprintf("💬 Chat: `%s`", v.Info.Chat.String()))

	return ReplyErr(v, strings.Join(lines, "\n"))
}

// .copy  — forward/copy the replied message back as a new message
func copyMsgHandler(v *events.Message, args []string) error {
	ext := v.Message.GetExtendedTextMessage()
	if ext == nil {
		return ReplyErr(v, "❌ Reply to a message to copy it.")
	}

	ci := ext.GetContextInfo()
	if ci == nil || ci.GetQuotedMessage() == nil {
		return ReplyErr(v, "❌ No quoted message found.")
	}

	qm := ci.GetQuotedMessage()

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := wac.SendMessage(ctx, v.Info.Chat, qm)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Copy failed: %v", err))
	}
	return nil
}

func init() {
	Register(ModuleInfo{
		Name:        "Message Tools",
		Description: "View quoted message content, message info, copy messages",
		Commands: []CommandInfo{
			{Pattern: "quoted", Handler: quotedHandler, Sudo: true},
			{Pattern: "msginfo", Handler: msgInfoHandler, Sudo: true},
			{Pattern: "copy", Handler: copyMsgHandler, Sudo: true},
		},
	})
}
