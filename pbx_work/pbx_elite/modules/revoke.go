package modules

import (
	"context"
	"fmt"
	"time"


	"go.mau.fi/whatsmeow/types/events"
)

// .revoke   — delete/revoke the replied-to message (bot must have sent it, or be group admin)
// .del      — alias for .revoke

func revokeHandler(v *events.Message, args []string) error {
	// Find the message to revoke via quoted message context
	var targetID string
	var targetSenderJID string
	var fromMe bool

	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		if ci := ext.GetContextInfo(); ci != nil {
			if ci.StanzaID != nil {
				targetID = *ci.StanzaID
			}
			if ci.Participant != nil {
				targetSenderJID = *ci.Participant
			}
			fromMe = ci.GetQuotedMessage() != nil && targetSenderJID == ""
		}
	}

	if targetID == "" {
		return ReplyErr(v, "❌ Reply to a message to delete/revoke it.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Determine if the bot sent the message being revoked
	if targetSenderJID == "" || (wac.Store.ID != nil && targetSenderJID == wac.Store.ID.String()) {
		fromMe = true
	}

	_, err := wac.RevokeMessage(ctx, v.Info.Chat, fromMe, targetID, "", "")
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Revoke failed: %v\n_Note: You can only revoke messages sent by the bot._", err))
	}

	return nil // silent success
}

func init() {
	Register(ModuleInfo{
		Name:        "Revoke",
		Description: "Delete/revoke a message (reply to target message)",
		Commands: []CommandInfo{
			{Pattern: "revoke", Handler: revokeHandler, Sudo: true},
			{Pattern: "del", Handler: revokeHandler, Sudo: true},
		},
	})
}
