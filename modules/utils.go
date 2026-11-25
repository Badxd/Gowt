package modules

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"PbxGb/client"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

// REPLY MESSAGE
func Reply(
	v *events.Message,
	text string,
) (*waProto.Message, error) {

	retries := 0

	for retries < 3 {

		_, err := client.Get().SendMessage(
			context.Background(),
			v.Info.Chat,
			&waProto.Message{
				ExtendedTextMessage: &waProto.ExtendedTextMessage{
					Text: &text,
					ContextInfo: &waProto.ContextInfo{
						StanzaID:      &v.Info.ID,
						Participant:   strPtr(v.Info.Sender.String()),
						QuotedMessage: v.Message,
					},
				},
			},
		)

		if err != nil {

			retries++

			time.Sleep(
				time.Duration(retries*2) *
					time.Second,
			)

			continue
		}

		return nil, nil
	}

	return nil, fmt.Errorf(
		"reply failed after %d retries",
		retries,
	)
}

// REPLY ERROR
func ReplyErr(
	v *events.Message,
	text string,
) error {

	_, err := Reply(
		v,
		text,
	)

	return err
}

// SEND NORMAL TEXT
func SendText(
	v *events.Message,
	text string,
) {

	_, err := client.Get().SendMessage(
		context.Background(),
		v.Info.Chat,
		&waProto.Message{
			Conversation: &text,
		},
	)

	if err != nil {

		log.Printf(
			"SendText error: %v",
			err,
		)
	}
}

// GET COMMAND ARGS
func GetArgs(
	v *events.Message,
) string {

	text := extractText(v)

	if idx := strings.Index(
		text,
		" ",
	); idx != -1 {

		return strings.TrimSpace(
			text[idx+1:],
		)
	}

	return ""
}

// CHECK REPLY
func IsReply(
	v *events.Message,
) bool {

	if v.Message == nil ||
		v.Message.ExtendedTextMessage == nil {

		return false
	}

	ctx := v.Message.
		ExtendedTextMessage.
		ContextInfo

	return ctx != nil &&
		ctx.QuotedMessage != nil
}

// GET QUOTED MESSAGE
func GetQuotedMessage(
	v *events.Message,
) *waProto.Message {

	if !IsReply(v) {
		return nil
	}

	return v.Message.
		ExtendedTextMessage.
		ContextInfo.
		QuotedMessage
}

// SENDER JID
func SenderJID(
	v *events.Message,
) string {

	return v.Info.Sender.String()
}

// STRING POINTER
func strPtr(
	s string,
) *string {

	return &s
}
