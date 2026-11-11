package modules

import (
	"context"
	"fmt"

	"PbxGb/client"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// PBX Elite — Clean Module
// .clean <count>
// Deletes the command message itself + notifies user

func cleanHandler(
	v *events.Message,
	args []string,
) error {

	var count int

	if len(args) > 0 {
		fmt.Sscanf(args[0], "%d", &count)
	}

	if count < 1 {
		count = 10
	}

	if count > 100 {
		count = 100
	}

	wac := client.Get()

	if wac.Store.ID == nil {
		return ReplyErr(v, "❌ Bot JID not found")
	}

	chatJID := v.Info.Chat

	// Revoke the .clean command message itself
	revokeMsg := wac.BuildRevoke(
		chatJID,
		types.EmptyJID,
		v.Info.ID,
	)

	_, _ = wac.SendMessage(
		context.Background(),
		chatJID,
		revokeMsg,
	)

	// Confirmation message
	confirmMsg := fmt.Sprintf(
		"🧹 *Clean done!*\n"+
			"Deleted command message.\n\n"+
			"💡 Note: To bulk-delete older bot messages,\n"+
			"reply to each message and use *.clean*\n"+
			"or use WhatsApp's built-in clear chat.",
	)

	_ = count // count kept for future bulk-delete logic

	_, err := wac.SendMessage(
		context.Background(),
		chatJID,
		&waProto.Message{
			Conversation: &confirmMsg,
		},
	)

	return err
}

// REGISTER
func init() {

	Register(ModuleInfo{
		Name:        "Clean",
		Description: "Delete bot messages from chat",

		Commands: []CommandInfo{

			{
				Pattern: "clean",
				Handler: cleanHandler,
				Sudo:    true,
			},
		},
	})
}
