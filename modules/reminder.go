package modules

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"PbxGb/client"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

// .remind <minutes> <message>
func remindHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		return ReplyErr(v, "Usage: .remind <minutes> <message>\nExample: .remind 10 Call mom")
	}

	mins, err := strconv.Atoi(args[0])
	if err != nil || mins <= 0 {
		return ReplyErr(v, "❌ Invalid minutes. Use a positive number.")
	}
	if mins > 1440 { // max 24h
		mins = 1440
	}

	msg := strings.Join(args[1:], " ")
	chat := v.Info.Chat
	msgID := v.Info.ID
	sender := v.Info.Sender.String()
	originalMsg := v.Message

	duration := time.Duration(mins) * time.Minute

	go func() {
		time.Sleep(duration)
		wac := client.Get()
		reminder := fmt.Sprintf("⏰ *Reminder!*\n\n_%s_", msg)
		_, _ = wac.SendMessage(context.Background(), chat, &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: &reminder,
				ContextInfo: &waProto.ContextInfo{
					StanzaID:      &msgID,
					Participant:   strPtr(sender),
					QuotedMessage: originalMsg,
				},
			},
		})
	}()

	readableTime := time.Now().Add(duration).Format("15:04:05")
	return ReplyErr(v, fmt.Sprintf("✅ Reminder set!\n⏰ Will ping at *%s*\n📝 _%s_", readableTime, msg))
}

func init() {
	Register(ModuleInfo{
		Name:        "Reminder",
		Description: "Set a timed reminder in any chat",
		Commands: []CommandInfo{
			{Pattern: "remind", Handler: remindHandler, Sudo: true},
		},
	})
}
