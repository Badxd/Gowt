package modules

import (
	"context"
	"fmt"
	"strings"
	"time"


	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// .number <@mention>  — show phone number of mentioned user
// .checkwa <number>   — check if a number is on WhatsApp
// .myid               — show your own WhatsApp JID
// .chatid             — show current chat JID

func numberHandler(v *events.Message, args []string) error {
	var targetJID types.JID

	// from mention in quoted message
	if v.Message.GetExtendedTextMessage() != nil {
		ctx := v.Message.GetExtendedTextMessage().GetContextInfo()
		if ctx != nil && len(ctx.GetMentionedJID()) > 0 {
			jid, err := types.ParseJID(ctx.GetMentionedJID()[0])
			if err == nil {
				targetJID = jid
			}
		}
	}

	if targetJID.IsEmpty() {
		if len(args) > 0 {
			num := strings.TrimPrefix(args[0], "+")
			num = strings.TrimPrefix(num, "@")
			jid, err := types.ParseJID(num + "@s.whatsapp.net")
			if err == nil {
				targetJID = jid
			}
		}
	}

	if targetJID.IsEmpty() {
		return ReplyErr(v, "Usage: .number @mention or .number <phone>")
	}

	return ReplyErr(v, fmt.Sprintf("📱 *Number:* +%s", targetJID.User))
}

func checkWAHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .checkwa <number> (e.g. .checkwa 917012345678)")
	}

	num := strings.TrimPrefix(args[0], "+")
	jid, err := types.ParseJID(num + "@s.whatsapp.net")
	if err != nil {
		return ReplyErr(v, "❌ Invalid number format.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := wac.GetUserInfo([]types.JID{jid})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Check failed: %v", err))
	}

	if info, ok := result[jid]; ok && info.VerifiedName != nil {
		return ReplyErr(v, fmt.Sprintf("✅ *+%s* is on WhatsApp!\n📝 Status: %s", num, info.Status))
	} else if _, ok := result[jid]; ok {
		return ReplyErr(v, fmt.Sprintf("✅ *+%s* is on WhatsApp!", num))
	}

	return ReplyErr(v, fmt.Sprintf("❌ *+%s* is NOT on WhatsApp.", num))
}

func myIDHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	own := wac.Store.ID
	if own == nil {
		return ReplyErr(v, "❌ Could not get own JID.")
	}
	return ReplyErr(v, fmt.Sprintf("🆔 *Your JID:* %s\n📱 *Number:* +%s", own.String(), own.User))
}

func chatIDHandler(v *events.Message, args []string) error {
	return ReplyErr(v, fmt.Sprintf("💬 *Chat JID:* `%s`", v.Info.Chat.String()))
}

func init() {
	Register(ModuleInfo{
		Name:        "Contacts",
		Description: "Number lookup, WhatsApp check, JID tools",
		Commands: []CommandInfo{
			{Pattern: "number", Handler: numberHandler, Sudo: true},
			{Pattern: "checkwa", Handler: checkWAHandler, Sudo: true},
			{Pattern: "myid", Handler: myIDHandler, Sudo: true},
			{Pattern: "chatid", Handler: chatIDHandler, Sudo: true},
		},
	})
}
