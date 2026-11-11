package modules

// reaction_helper.go — builds WA reaction message proto

import (
	waProto "go.mau.fi/whatsmeow/binary/proto"
)

// buildReaction builds a WhatsApp reaction message
func buildReaction(
	msgID string,
	senderJID string,
	emoji string,
) *waProto.Message {

	return &waProto.Message{
		ReactionMessage: &waProto.ReactionMessage{
			Key: &waProto.MessageKey{
				RemoteJID:   &senderJID,
				FromMe:      boolPtr(false),
				ID:          &msgID,
			},
			Text:              &emoji,
			SenderTimestampMS: int64Ptr(0),
		},
	}
}


func int64Ptr(i int64) *int64 { return &i }
