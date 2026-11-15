package modules

import (
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

	msgID := quoted.GetStanzaID()
	if msgID == "" {
		return ReplyErr(v, "❌ Could not get message ID from quoted context.")
	}

	return ReplyErr(v, "❌ Pin message is not supported in this version.")
}

func unpinHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	return ReplyErr(v, "❌ Unpin message is not supported in this version.")
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
