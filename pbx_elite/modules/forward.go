package modules

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// .fwd <jid|number>  — forward quoted message to another chat/number
// .broadcast <msg>   — send a message to all saved contacts (sudo only)

func fwdHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .fwd <number or JID>\nReply to a message to forward it.")
	}

	// get the message to forward
	var fwdMsg interface{}
	if v.Message.GetExtendedTextMessage() != nil {
		ctx := v.Message.GetExtendedTextMessage().GetContextInfo()
		if ctx != nil && ctx.GetQuotedMessage() != nil {
			fwdMsg = ctx.GetQuotedMessage()
		}
	}
	if fwdMsg == nil {
		return ReplyErr(v, "❌ Reply to a message you want to forward.")
	}

	target := args[0]
	// normalize: if pure digits, add @s.whatsapp.net
	if !strings.Contains(target, "@") {
		target = target + "@s.whatsapp.net"
	}

	targetJID, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Invalid JID/number: %v", err))
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	// forward as the quoted message proto
	switch msg := fwdMsg.(type) {
	case interface{ ProtoReflect() interface{} }:
		_ = msg
	default:
		_ = msg
	}

	// send the quoted message content as-is
	quoted := v.Message.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
	if quoted == nil {
		return ReplyErr(v, "❌ Could not extract quoted message.")
	}

	_, err = wac.SendMessage(context.Background(), targetJID, quoted)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Forward failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("✅ Message forwarded to %s", targetJID.User))
}

func init() {
	Register(ModuleInfo{
		Name:        "Forward",
		Description: "Forward quoted messages to other chats/numbers",
		Commands: []CommandInfo{
			{Pattern: "fwd", Handler: fwdHandler, Sudo: true},
		},
	})
}
