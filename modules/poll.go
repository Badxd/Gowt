package modules

import (
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

// Poll: create a WhatsApp poll in a group or DM.
//
// .poll <question> | <opt1> | <opt2> [| opt3...]
// .poll "What's for lunch?" | Pizza | Burger | Salad
//
// WhatsApp supports 2–12 options.
// Uses wac.BuildPollCreation() which is the proper whatsmeow API.

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

	// BuildPollCreation is the official whatsmeow API for polls.
	// selectableOptionCount=1 means single-choice.
	pollMsg := wac.BuildPollCreation(question, options, 1)

	ctx, cancel := newContext(15)
	defer cancel()

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

	for _, o := range options {
		if o == "" {
			return ReplyErr(v, "❌ Poll options cannot be empty.")
		}
	}

	// selectableOptionCount=0 means unlimited / multiple choice
	pollMsg := wac.BuildPollCreation(question, options, 0)

	ctx, cancel := newContext(15)
	defer cancel()

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
