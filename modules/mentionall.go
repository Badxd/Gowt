package modules

import (
	"context"
	"fmt"
	"strings"
	"time"


	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// .hidetag [msg] — send a message that silently pings everyone (hidden tag)
// already exists as "tag" in grouptools.go, but many bots expose it as .hidetag too
// .kickme        — leave group yourself
// .leave         — bot leaves the group (sudo)


func kickMeHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ Groups only.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	senderJID := v.Info.Sender
	toRemove := []types.JID{senderJID}
	_, err := wac.UpdateGroupParticipants(ctx, v.Info.Chat, toRemove, "remove")
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Could not kick you: %v", err))
	}
	return nil
}


func init() {
	Register(ModuleInfo{
		Name:        "Mention All",
		Description: "Hidden tag all members, kickme, leave group",
		Commands: []CommandInfo{
			{Pattern: "kickme", Handler: kickMeHandler, Sudo: false},
		},
	})
}
