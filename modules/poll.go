package modules

import (
	"fmt"
	"strings"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// Poll: create a WhatsApp poll in a group or DM.
//
// .poll <question> | <opt1> | <opt2> [| opt3...]
// .poll "What's for lunch?" | Pizza | Burger | Salad
//
// WhatsApp supports 2–12 options.

func pollHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	if len(args) == 0 {
		return ReplyErr(v, "Usage: .poll <question> | <option1> | <option2> [| option3...]\nExample: .poll Favourite fruit? | Mango | Apple | Banana")
	}

	full := strings.Join(args, " ")
	parts := strings.Split(full, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	if len(parts) < 3 {
		return ReplyErr(v, "❌ Need at least a question and 2 options.\nFormat: .poll Question | Option 1 | Option 2")
	}
	if len(parts) > 13 {
		return ReplyErr(v, "❌ Maximum 12 options allowed.")
	}

	question := parts[0]
	options := parts[1:]

	if question == "" {
		return ReplyErr(v, "❌ Poll question cannot be empty.")
	}
	for _, o := range options {
		if o == "" {
			return ReplyErr(v, "❌ Poll options cannot be empty.")
		}
	}

	// Build poll options
	pollOptions := make([]*waProto.PollCreationMessage_Option, len(options))
	for i, o := range options {
		pollOptions[i] = &waProto.PollCreationMessage_Option{
			OptionName: proto.String(o),
		}
	}

	ctx, cancel := newContext(15)
	defer cancel()

	pollMsg := &waProto.Message{
		PollCreationMessage: &waProto.PollCreationMessage{
			Name:                proto.String(question),
			Options:             pollOptions,
			SelectableOptionsCount: proto.Uint32(1), // single choice; 0 = multiple
		},
	}

	_, err := wac.SendMessage(ctx, v.Info.Chat, pollMsg)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to send poll: %v", err))
	}
	return nil
}

func multipollHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	if len(args) == 0 {
		return ReplyErr(v, "Usage: .multipoll <question> | <opt1> | <opt2> [...]\nSame as .poll but allows multiple selections.")
	}

	full := strings.Join(args, " ")
	parts := strings.Split(full, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	if len(parts) < 3 {
		return ReplyErr(v, "❌ Need at least a question and 2 options.")
	}
	if len(parts) > 13 {
		return ReplyErr(v, "❌ Maximum 12 options allowed.")
	}

	question := parts[0]
	options := parts[1:]

	pollOptions := make([]*waProto.PollCreationMessage_Option, len(options))
	for i, o := range options {
		pollOptions[i] = &waProto.PollCreationMessage_Option{
			OptionName: proto.String(o),
		}
	}

	ctx, cancel := newContext(15)
	defer cancel()

	// SelectableOptionsCount = 0 → unlimited (multiple choice)
	pollMsg := &waProto.Message{
		PollCreationMessage: &waProto.PollCreationMessage{
			Name:                   proto.String(question),
			Options:                pollOptions,
			SelectableOptionsCount: proto.Uint32(0),
		},
	}

	_, err := wac.SendMessage(ctx, v.Info.Chat, pollMsg)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to send poll: %v", err))
	}
	return nil
}

func init() {
	Register(ModuleInfo{
		Name:        "Poll",
		Description: "Create WhatsApp polls in groups or DMs",
		Commands: []CommandInfo{
			{Pattern: "poll", Handler: pollHandler, Sudo: true},
			{Pattern: "multipoll", Handler: multipollHandler, Sudo: true},
		},
	})
}
