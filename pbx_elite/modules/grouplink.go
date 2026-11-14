package modules

import (
	"context"
	"fmt"
	"strings"
	"time"


	"go.mau.fi/whatsmeow/types/events"
)

// .glink  — get group invite link
// .revoke — revoke + regenerate group invite link

func glinkHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command only works in groups.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	link, err := wac.GetGroupInviteLink(ctx, v.Info.Chat, false)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to get invite link: %v", err))
	}

	return ReplyErr(v, fmt.Sprintf("🔗 *Group Invite Link*\nhttps://chat.whatsapp.com/%s", link))
}

func revokeInviteHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command only works in groups.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// revoke old link
	if err := wac.RevokeGroupInviteLink(ctx, v.Info.Chat); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to revoke invite link: %v", err))
	}

	// fetch new link
	newLink, err := wac.GetGroupInviteLink(ctx, v.Info.Chat, false)
	if err != nil {
		return ReplyErr(v, "✅ Old link revoked. Could not fetch new link — try .glink")
	}

	return ReplyErr(v, fmt.Sprintf("♻️ *Invite link revoked & regenerated*\nhttps://chat.whatsapp.com/%s", newLink))
}

func init() {
	Register(ModuleInfo{
		Name:        "Group Link",
		Description: "Get or revoke/reset group invite link",
		Commands: []CommandInfo{
			{Pattern: "glink", Handler: glinkHandler, Sudo: true},
		},
	})
}
