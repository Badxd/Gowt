package modules

// Poll Tools — create WhatsApp polls in groups or DMs.
//
// Commands:
//   .poll <question> | <opt1> | <opt2> [| opt3...]   — create a poll (up to 12 options)
//   .pollmulti <question> | <opt1> | <opt2> [| ...]  — multi-select poll
//   .quickpoll <question>                            — yes/no quick poll

import (
	"context"
	"fmt"
	"strings"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"PbxGb/client"
)

// ── CREATE POLL ───────────────────────────────────────────────────────────────
func pollHandler(v *events.Message, args []string, multiSelect bool) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .poll <question> | <option1> | <option2> [| more...]")
	}

	raw := strings.Join(args, " ")
	parts := strings.Split(raw, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	if len(parts) < 3 {
		return ReplyErr(v, "❌ Need at least a question and 2 options.\nExample: .poll Best fruit? | Apple | Mango | Banana")
	}

	question := parts[0]
	options := parts[1:]

	if len(options) > 12 {
		options = options[:12] // WhatsApp cap
	}

	// Build PollOption list
	pollOptions := make([]*waProto.PollCreationMessage_Option, 0, len(options))
	for _, opt := range options {
		if opt == "" {
			continue
		}
		pollOptions = append(pollOptions, &waProto.PollCreationMessage_Option{
			OptionName: proto.String(opt),
		})
	}

	if len(pollOptions) < 2 {
		return ReplyErr(v, "❌ Need at least 2 non-empty options.")
	}

	// selectableCount: 0 = multi-select, 1 = single choice
	selectableCount := uint32(1)
	if multiSelect {
		selectableCount = 0
	}

	wac := client.Get()
	ctx := context.Background()

	_, err := wac.SendMessage(ctx, v.Info.Chat, &waProto.Message{
		PollCreationMessage: &waProto.PollCreationMessage{
			Name:                 proto.String(question),
			Options:              pollOptions,
			SelectableOptionsCount: proto.Uint32(selectableCount),
		},
	})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Poll creation failed: %v", err))
	}
	return nil
}

func singlePollHandler(v *events.Message, args []string) error {
	return pollHandler(v, args, false)
}

func multiPollHandler(v *events.Message, args []string) error {
	return pollHandler(v, args, true)
}

// ── QUICK YES/NO POLL ─────────────────────────────────────────────────────────
func quickPollHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .quickpoll <question>")
	}
	question := strings.Join(args, " ")
	wac := client.Get()
	ctx := context.Background()

	_, err := wac.SendMessage(ctx, v.Info.Chat, &waProto.Message{
		PollCreationMessage: &waProto.PollCreationMessage{
			Name: proto.String(question),
			Options: []*waProto.PollCreationMessage_Option{
				{OptionName: proto.String("✅ Yes")},
				{OptionName: proto.String("❌ No")},
			},
			SelectableOptionsCount: proto.Uint32(1),
		},
	})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Poll failed: %v", err))
	}
	return nil
}

func init() {
	Register(ModuleInfo{
		Name:        "PollTools",
		Description: "Create single/multi-select polls and quick yes/no polls",
		Commands: []CommandInfo{
			{Pattern: "poll",       Handler: singlePollHandler, Sudo: true},
			{Pattern: "pollmulti",  Handler: multiPollHandler,  Sudo: true},
			{Pattern: "quickpoll",  Handler: quickPollHandler,  Sudo: true},
			{Pattern: "yesnopoll",  Handler: quickPollHandler,  Sudo: true},
		},
	})
}
