package modules

// React Tools — send emoji reactions to messages.
//
// Commands:
//   .react <emoji>        — react to quoted/replied message with emoji
//   .unreact              — remove your reaction from quoted message
//   .react2 <emoji>       — alias for .react

import (
	"context"
	"fmt"
	"strings"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"PbxGb/client"
)

// getReactionTarget extracts the key (JID + message ID) of the message
// we're reacting to — either the message we're replying to, or the
// incoming message itself if no context info is present.
func getReactionTarget(v *events.Message) (senderJID string, msgID string, ok bool) {
	if v.Message == nil {
		return "", "", false
	}

	// Check for quoted message via ExtendedTextMessage
	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		ci := ext.GetContextInfo()
		if ci != nil && ci.StanzaId != nil {
			participant := ci.GetParticipant()
			if participant == "" {
				participant = ci.GetRemoteJid()
			}
			return participant, ci.GetStanzaId(), true
		}
	}

	// Fall back to reacting on any message type's context
	getCI := func(ci *waProto.ContextInfo) (string, string, bool) {
		if ci == nil || ci.StanzaId == nil {
			return "", "", false
		}
		p := ci.GetParticipant()
		if p == "" {
			p = ci.GetRemoteJid()
		}
		return p, ci.GetStanzaId(), true
	}

	if img := v.Message.GetImageMessage(); img != nil {
		return getCI(img.GetContextInfo())
	}
	if vid := v.Message.GetVideoMessage(); vid != nil {
		return getCI(vid.GetContextInfo())
	}
	if aud := v.Message.GetAudioMessage(); aud != nil {
		return getCI(aud.GetContextInfo())
	}
	if doc := v.Message.GetDocumentMessage(); doc != nil {
		return getCI(doc.GetContextInfo())
	}

	// If no quoted context, react to the incoming message itself
	return v.Info.Sender.String(), v.Info.ID, true
}

// ── REACT ──────────────────────────────────────────────────────────────────
func reactHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .react <emoji>  (reply to a message)")
	}
	emoji := strings.TrimSpace(args[0])

	senderJID, msgID, ok := getReactionTarget(v)
	if !ok {
		return ReplyErr(v, "❌ Could not find target message. Reply to a message first.")
	}

	wac := client.Get()
	ctx := context.Background()

	chatStr := v.Info.Chat.String()

	_, err := wac.SendMessage(ctx, v.Info.Chat, &waProto.Message{
		ReactionMessage: &waProto.ReactionMessage{
			Key: &waProto.MessageKey{
				RemoteJid:   proto.String(chatStr),
				FromMe:      proto.Bool(senderJID == "" || senderJID == wac.Store.ID.String()),
				Id:          proto.String(msgID),
				Participant: proto.String(senderJID),
			},
			Text:              proto.String(emoji),
			SenderTimestampMs: proto.Int64(v.Info.Timestamp.UnixMilli()),
		},
	})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Reaction failed: %v", err))
	}
	return nil
}

// ── UNREACT ────────────────────────────────────────────────────────────────
func unreactHandler(v *events.Message, args []string) error {
	senderJID, msgID, ok := getReactionTarget(v)
	if !ok {
		return ReplyErr(v, "❌ Could not find target message. Reply to the message first.")
	}

	wac := client.Get()
	ctx := context.Background()

	chatStr := v.Info.Chat.String()

	// Empty text = remove reaction
	_, err := wac.SendMessage(ctx, v.Info.Chat, &waProto.Message{
		ReactionMessage: &waProto.ReactionMessage{
			Key: &waProto.MessageKey{
				RemoteJid:   proto.String(chatStr),
				FromMe:      proto.Bool(senderJID == "" || senderJID == wac.Store.ID.String()),
				Id:          proto.String(msgID),
				Participant: proto.String(senderJID),
			},
			Text:              proto.String(""),
			SenderTimestampMs: proto.Int64(v.Info.Timestamp.UnixMilli()),
		},
	})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Remove reaction failed: %v", err))
	}
	return nil
}

func init() {
	Register(ModuleInfo{
		Name:        "ReactTools",
		Description: "Send and remove emoji reactions on messages",
		Commands: []CommandInfo{
			{Pattern: "react",   Handler: reactHandler,   Sudo: true},
			{Pattern: "react2",  Handler: reactHandler,   Sudo: true},
			{Pattern: "unreact", Handler: unreactHandler, Sudo: true},
		},
	})
}
