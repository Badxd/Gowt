package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"PbxGb/client"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func gcastHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .gcast <message>\nBroadcasts a message to all groups.")
	}
	msg := strings.Join(args, " ")
	wac := client.Get()

	chats, err := wac.Store.Contacts.GetAllContacts()
	if err != nil {
		return ReplyErr(v, "❌ Failed to fetch contacts: "+err.Error())
	}

	sent := 0
	failed := 0
	for jid := range chats {
		if jid.Server != types.GroupServer {
			continue
		}
		m := msg
		_, sendErr := wac.SendMessage(context.Background(), jid, &waProto.Message{
			Conversation: &m,
		})
		if sendErr != nil {
			failed++
		} else {
			sent++
		}
		time.Sleep(500 * time.Millisecond) // flood protection
	}

	return ReplyErr(v, fmt.Sprintf("📢 Broadcast done!\n✅ Sent: %d\n❌ Failed: %d", sent, failed))
}

func gcastUsersHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .gcastusers <message>\nBroadcasts a message to all private chats.")
	}
	msg := strings.Join(args, " ")
	wac := client.Get()

	chats, err := wac.Store.Contacts.GetAllContacts()
	if err != nil {
		return ReplyErr(v, "❌ Failed to fetch contacts: "+err.Error())
	}

	sent := 0
	failed := 0
	for jid := range chats {
		if jid.Server != types.DefaultUserServer {
			continue
		}
		m := msg
		_, sendErr := wac.SendMessage(context.Background(), jid, &waProto.Message{
			Conversation: &m,
		})
		if sendErr != nil {
			failed++
		} else {
			sent++
		}
		time.Sleep(500 * time.Millisecond)
	}

	return ReplyErr(v, fmt.Sprintf("📢 User broadcast done!\n✅ Sent: %d\n❌ Failed: %d", sent, failed))
}

func init() {
	Register(ModuleInfo{
		Name:        "Broadcast",
		Description: "Broadcast messages to all groups or users",
		Commands: []CommandInfo{
			{Pattern: "gcast", Handler: gcastHandler, Sudo: true},
			{Pattern: "gcastusers", Handler: gcastUsersHandler, Sudo: true},
		},
	})
}
