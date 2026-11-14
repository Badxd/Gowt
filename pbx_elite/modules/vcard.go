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

// .vcard <number>  — send contact card for a number
// .vcardme         — send your own contact card

func buildVCard(name, phone string) string {
	return fmt.Sprintf(
		"BEGIN:VCARD\r\n"+
			"VERSION:3.0\r\n"+
			"FN:%s\r\n"+
			"TEL;TYPE=CELL:+%s\r\n"+
			"END:VCARD",
		name, phone,
	)
}

func vcardHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .vcard <number> [name]\nExample: .vcard 917012345678 John Doe")
	}

	num := strings.TrimPrefix(args[0], "+")
	name := "Contact"
	if len(args) > 1 {
		name = strings.Join(args[1:], " ")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	vcard := buildVCard(name, num)

	displayName := name
	msg := &waE2E.Message{
		ContactMessage: &waE2E.ContactMessage{
			DisplayName: &displayName,
			Vcard:       &vcard,
		},
	}

	_, err := wac.SendMessage(ctx, v.Info.Chat, msg)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Send failed: %v", err))
	}
	return nil
}

func vcardMeHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	own := wac.Store.ID
	if own == nil {
		return ReplyErr(v, "❌ Could not get own JID.")
	}

	name := wac.Store.PushName
	if name == "" {
		name = own.User
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	vcard := buildVCard(name, own.User)
	displayName := name

	msg := &waE2E.Message{
		ContactMessage: &waE2E.ContactMessage{
			DisplayName: &displayName,
			Vcard:       &vcard,
		},
	}

	_, err := wac.SendMessage(ctx, v.Info.Chat, msg)
	return err
}

func vcardMentionHandler(v *events.Message, args []string) error {
	// .vcardtag @mention — send vcard for a mentioned person
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	var targetJID types.JID
	var name string

	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		ci := ext.GetContextInfo()
		if ci != nil && len(ci.GetMentionedJID()) > 0 {
			jid, err := types.ParseJID(ci.GetMentionedJID()[0])
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
		return ReplyErr(v, "Usage: .vcardtag @mention or .vcardtag <number>")
	}

	// try to get their push name
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	info, err := wac.GetUserInfo(ctx, []types.JID{targetJID})
	if err == nil {
		if ui, ok := info[targetJID]; ok {
			name = ui.PushName
		}
	}
	if name == "" {
		name = targetJID.User
	}

	vcard := buildVCard(name, targetJID.User)
	displayName := name

	msg := &waE2E.Message{
		ContactMessage: &waE2E.ContactMessage{
			DisplayName: &displayName,
			Vcard:       &vcard,
		},
	}

	_, err = wac.SendMessage(ctx, v.Info.Chat, msg)
	return err
}

func init() {
	Register(ModuleInfo{
		Name:        "vCard",
		Description: "Send WhatsApp contact cards",
		Commands: []CommandInfo{
			{Pattern: "vcard", Handler: vcardHandler, Sudo: true},
			{Pattern: "vcardme", Handler: vcardMeHandler, Sudo: true},
			{Pattern: "vcardtag", Handler: vcardMentionHandler, Sudo: true},
		},
	})
}
