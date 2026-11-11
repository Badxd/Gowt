package modules

import (
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

// .pin    — pin the quoted message in the group
// .unpin  — unpin the quoted message (or all if no quote)

func pinHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	quoted := v.Message.GetExtendedTextMessage().GetContextInfo()
	if quoted == nil || quoted.GetQuotedMessage() == nil {
		return ReplyErr(v, "❌ Reply to a message to pin it.")
	}

	msgID := quoted.GetStanzaId()
	if msgID == "" {
		return ReplyErr(v, "❌ Could not get message ID from quoted context.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	ctx, cancel := newContext(15)
	defer cancel()

	// duration: 0 = unpin, 86400 = 24h, 604800 = 7d, 2592000 = 30d
	// default pin for 7 days
	duration := uint32(604800)
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "24h", "1d":
			duration = 86400
		case "7d", "week":
			duration = 604800
		case "30d", "month":
			duration = 2592000
		}
	}

	err := wac.UpdatePinnedMessages(ctx, v.Info.Chat, msgID, true, duration)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to pin message: %v", err))
	}

	durationStr := "7 days"
	switch duration {
	case 86400:
		durationStr = "24 hours"
	case 2592000:
		durationStr = "30 days"
	}
	return ReplyErr(v, fmt.Sprintf("📌 Message pinned for *%s*.", durationStr))
}

func unpinHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	quoted := v.Message.GetExtendedTextMessage().GetContextInfo()
	if quoted == nil || quoted.GetQuotedMessage() == nil {
		return ReplyErr(v, "❌ Reply to the pinned message to unpin it.")
	}

	msgID := quoted.GetStanzaId()
	if msgID == "" {
		return ReplyErr(v, "❌ Could not get message ID from quoted context.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	ctx, cancel := newContext(15)
	defer cancel()

	// duration 0 = unpin
	err := wac.UpdatePinnedMessages(ctx, v.Info.Chat, msgID, false, 0)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to unpin message: %v", err))
	}

	return ReplyErr(v, "📌 Message unpinned.")
}

func init() {
	Register(ModuleInfo{
		Name:        "Pin",
		Description: "Pin or unpin messages in a group",
		Commands: []CommandInfo{
			{Pattern: "pin", Handler: pinHandler, Sudo: true},
			{Pattern: "unpin", Handler: unpinHandler, Sudo: true},
		},
	})
}
