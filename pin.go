package modules

// .pin   — pin the quoted/replied message in the group
// .unpin — unpin the quoted/replied message in the group
//
// WhatsApp pin protocol: send a ProtocolMessage with PinInChatMessage sub-field.
// wac.PinMessage() does NOT exist in whatsmeow — this is the correct approach.

import (
	"context"
	"fmt"
	"strings"
	"time"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func pinOrUnpin(v *events.Message, pin bool) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	quoted := v.Message.GetExtendedTextMessage().GetContextInfo()
	if quoted == nil || quoted.GetStanzaID() == "" {
		return ReplyErr(v, "❌ Reply to a message to pin/unpin it.")
	}

	msgID := quoted.GetStanzaID()
	participant := quoted.GetParticipant()

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	// Determine fromMe: compare quoted participant to self JID
	selfJID := getSelfJID()
	fromMe := false
	if participant != "" {
		qJID, err := types.ParseJID(participant)
		if err == nil {
			fromMe = qJID.User == selfJID.User
		}
	}

	pinType := waProto.PinInChatMessage_PIN_FOR_ALL
	if !pin {
		pinType = waProto.PinInChatMessage_UNPIN_FOR_ALL
	}

	pinMsg := &waProto.Message{
		PinInChatMessage: &waProto.PinInChatMessage{
			Key: &waProto.MessageKey{
				RemoteJID: proto.String(v.Info.Chat.String()),
				FromMe:    proto.Bool(fromMe),
				ID:        proto.String(msgID),
			},
			Type:              pinType.Enum(),
			SenderTimestampMS: proto.Int64(time.Now().UnixMilli()),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := wac.SendMessage(ctx, v.Info.Chat, pinMsg)
	if err != nil {
		if pin {
			return ReplyErr(v, fmt.Sprintf("❌ Pin failed: %v\n\nMake sure the bot is a group admin.", err))
		}
		return ReplyErr(v, fmt.Sprintf("❌ Unpin failed: %v", err))
	}

	if pin {
		return ReplyErr(v, "📌 *Message Pinned!*")
	}
	return ReplyErr(v, "📌 *Message Unpinned!*")
}

func pinHandler(v *events.Message, args []string) error {
	return pinOrUnpin(v, true)
}

func unpinHandler(v *events.Message, args []string) error {
	return pinOrUnpin(v, false)
}

func init() {
	Register(ModuleInfo{
		Name:        "Pin",
		Description: "Pin or unpin messages in a group (bot must be admin)",
		Commands: []CommandInfo{
			{Pattern: "pin",   Handler: pinHandler,   Sudo: true},
			{Pattern: "unpin", Handler: unpinHandler, Sudo: true},
		},
	})
}
