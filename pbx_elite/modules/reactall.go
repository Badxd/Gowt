package modules

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf16"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// .reactall <emoji>   — react to last N messages in current chat with emoji
// .react <emoji>      — react to the replied message with emoji

var defaultReactEmoji = "❤️"

// sendReaction sends a single reaction to a specific message
func sendReaction(wac *whatsmeow.Client, chat types.JID, msgID types.MessageID, senderJID types.JID, emoji string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// build the reaction key
	fromMe := senderJID.String() == wac.Store.ID.String()
	key := &waE2E.MessageKey{
		RemoteJID: proto.String(chat.String()),
		FromMe:    proto.Bool(fromMe),
		ID:        proto.String(string(msgID)),
	}
	if !fromMe && strings.Contains(chat.String(), "@g.us") {
		pStr := senderJID.String()
		key.Participant = proto.String(pStr)
	}

	reactionMsg := &waE2E.Message{
		ReactionMessage: &waE2E.ReactionMessage{
			Key:               key,
			Text:              proto.String(emoji),
			SenderTimestampMS: proto.Int64(time.Now().UnixMilli()),
		},
	}

	_, err := wac.SendMessage(ctx, chat, reactionMsg)
	return err
}

// encodeEmoji ensures the emoji is valid UTF-16 (WhatsApp requirement)
func encodeEmoji(s string) string {
	// Some emoji need surrogate pairs — round-trip through UTF-16
	r := []rune(s)
	if len(r) == 0 {
		return defaultReactEmoji
	}
	// just return first grapheme cluster
	u16 := utf16.Encode(r)
	return string(utf16.Decode(u16))
}

// reactHandler — react to a replied message
func reactHandler(v *events.Message, args []string) error {
	emoji := defaultReactEmoji
	if len(args) > 0 {
		emoji = encodeEmoji(strings.Join(args, ""))
	}

	// find the message to react to
	var targetID types.MessageID
	var targetSender types.JID

	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		if ci := ext.GetContextInfo(); ci != nil {
			if ci.StanzaID != nil {
				targetID = types.MessageID(*ci.StanzaID)
			}
			if ci.Participant != nil {
				j, err := types.ParseJID(*ci.Participant)
				if err == nil {
					targetSender = j
				}
			}
		}
	}

	if targetID == "" {
		return ReplyErr(v, "❌ Reply to a message to react to it.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	if err := sendReaction(wac, v.Info.Chat, targetID, targetSender, emoji); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ React failed: %v", err))
	}
	return nil
}

// reactAllHandler — react to last N messages in chat (fetched from store)
func reactAllHandler(v *events.Message, args []string) error {
	emoji := defaultReactEmoji
	if len(args) > 0 {
		emoji = encodeEmoji(strings.Join(args, ""))
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	// Get recent messages from store — whatsmeow stores message history
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msgs, err := wac.Store.Messages.GetMessagesBefore(ctx, v.Info.Chat, v.Info.ID, "", 25)
	if err != nil || len(msgs) == 0 {
		return ReplyErr(v, "❌ No messages found in history to react to.")
	}

	sent := 0
	failed := 0
	for _, m := range msgs {
		if m.IsFromMe {
			continue // skip own messages
		}
		senderJID, _ := types.ParseJID(m.SenderJID)
		err := sendReaction(wac, v.Info.Chat, types.MessageID(m.ID), senderJID, emoji)
		if err != nil {
			failed++
		} else {
			sent++
		}
		time.Sleep(300 * time.Millisecond) // rate-limit
	}

	return ReplyErr(v, fmt.Sprintf(
		"✅ Reacted %s to *%d* messages (%d failed).",
		emoji, sent, failed,
	))
}

func init() {
	Register(ModuleInfo{
		Name:        "React",
		Description: "React to messages with emoji",
		Commands: []CommandInfo{
			{Pattern: "react", Handler: reactHandler, Sudo: true},
			{Pattern: "reactall", Handler: reactAllHandler, Sudo: true},
		},
	})
}
