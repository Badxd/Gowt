package modules

import (
	"fmt"
	"strings"
	"time"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// .bc <message>        — broadcast message to ALL groups the bot is in
// .bcgroups            — list all groups bot is in
// .bcwarn <message>    — broadcast with a warning prefix
// .forward             — forward the quoted message to all groups

func broadcastHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .bc <message>")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	text := strings.Join(args, " ")

	groups, err := fetchAllGroups(wac)
	if err != nil || len(groups) == 0 {
		return ReplyErr(v, fmt.Sprintf("❌ Could not fetch groups: %v", err))
	}

	sent := 0
	failed := 0
	for _, g := range groups {
		ctx, cancel := newContext(10)
		msg := &waProto.Message{
			Conversation: proto.String(text),
		}
		_, sendErr := wac.SendMessage(ctx, g, msg)
		cancel()
		if sendErr != nil {
			failed++
		} else {
			sent++
		}
		time.Sleep(300 * time.Millisecond) // avoid rate-limit
	}

	return ReplyErr(v, fmt.Sprintf("📢 Broadcast complete.\n✅ Sent: %d\n❌ Failed: %d", sent, failed))
}

func bcWarnHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .bcwarn <message>")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	text := "⚠️ *IMPORTANT NOTICE*\n\n" + strings.Join(args, " ")

	groups, err := fetchAllGroups(wac)
	if err != nil || len(groups) == 0 {
		return ReplyErr(v, fmt.Sprintf("❌ Could not fetch groups: %v", err))
	}

	sent := 0
	failed := 0
	for _, g := range groups {
		ctx, cancel := newContext(10)
		msg := &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String(text),
			},
		}
		_, sendErr := wac.SendMessage(ctx, g, msg)
		cancel()
		if sendErr != nil {
			failed++
		} else {
			sent++
		}
		time.Sleep(300 * time.Millisecond)
	}

	return ReplyErr(v, fmt.Sprintf("⚠️ Warning broadcast complete.\n✅ Sent: %d\n❌ Failed: %d", sent, failed))
}

func bcGroupsHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	ctx, cancel := newContext(20)
	defer cancel()

	joined, err := wac.GetJoinedGroups()
	_ = ctx
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Could not fetch groups: %v", err))
	}

	if len(joined) == 0 {
		return ReplyErr(v, "ℹ️ Bot is not in any groups.")
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("📋 *Bot is in %d groups:*", len(joined)), "")
	for i, g := range joined {
		name := g.Name
		if name == "" {
			name = g.JID.User
		}
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, name))
		if i >= 49 {
			lines = append(lines, fmt.Sprintf("... and %d more", len(joined)-50))
			break
		}
	}

	return ReplyErr(v, strings.Join(lines, "\n"))
}

func forwardBroadcastHandler(v *events.Message, args []string) error {
	quoted := v.Message.GetExtendedTextMessage().GetContextInfo()
	if quoted == nil || quoted.GetQuotedMessage() == nil {
		return ReplyErr(v, "❌ Reply to a message to forward it to all groups.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	// Build forward message
	fwdMsg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(extractBodyText(v)),
			ContextInfo: &waProto.ContextInfo{
				IsForwarded:     proto.Bool(true),
				ForwardingScore: proto.Uint32(1),
			},
		},
	}

	if t := extractBodyText(v); t == "" {
		// just forward the quoted message itself
		fwdMsg = quoted.GetQuotedMessage()
	}

	groups, err := fetchAllGroups(wac)
	if err != nil || len(groups) == 0 {
		return ReplyErr(v, fmt.Sprintf("❌ Could not fetch groups: %v", err))
	}

	sent := 0
	failed := 0
	for _, g := range groups {
		ctx, cancel := newContext(10)
		_, sendErr := wac.SendMessage(ctx, g, fwdMsg)
		cancel()
		if sendErr != nil {
			failed++
		} else {
			sent++
		}
		time.Sleep(300 * time.Millisecond)
	}

	return ReplyErr(v, fmt.Sprintf("📤 Forward complete.\n✅ Sent: %d\n❌ Failed: %d", sent, failed))
}

// fetchAllGroups returns all group JIDs the bot is currently in.
func fetchAllGroups(wac interface {
	GetJoinedGroups() ([]*types.GroupInfo, error)
}) ([]types.JID, error) {
	groups, err := wac.GetJoinedGroups()
	if err != nil {
		return nil, err
	}
	jids := make([]types.JID, 0, len(groups))
	for _, g := range groups {
		jids = append(jids, g.JID)
	}
	return jids, nil
}

func init() {
	Register(ModuleInfo{
		Name:        "Broadcast",
		Description: "Send a message to all groups the bot is in",
		Commands: []CommandInfo{
			{Pattern: "bc", Handler: broadcastHandler, Sudo: true},
			{Pattern: "broadcast", Handler: broadcastHandler, Sudo: true},
			{Pattern: "bcwarn", Handler: bcWarnHandler, Sudo: true},
			{Pattern: "bcgroups", Handler: bcGroupsHandler, Sudo: true},
			{Pattern: "forward", Handler: forwardBroadcastHandler, Sudo: true},
		},
	})
}
