package modules

import (
	"context"
	"fmt"
	"strings"
	"time"


	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// .leave           — leave the current group
// .leaveall        — leave all groups (dangerous!)
// .joingroup <link> — join a group via invite link

func leaveGroupHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ Must be used inside a group.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// polite farewell first
	_ = ReplyErr(v, "👋 Goodbye!")
	time.Sleep(500 * time.Millisecond)

	err := wac.LeaveGroup(ctx, v.Info.Chat)
	if err != nil {
		return fmt.Errorf("leave failed: %w", err)
	}
	return nil
}

func leaveAllGroupsHandler(v *events.Message, args []string) error {
	// require confirmation argument to avoid accidental use
	if len(args) == 0 || strings.ToLower(args[0]) != "confirm" {
		return ReplyErr(v, "⚠️ This will make the bot leave ALL groups!\nTo confirm: `.leaveall confirm`")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	chats, err := wac.Store.Chats.GetAllChats()
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to get chats: %v", err))
	}

	left := 0
	failed := 0
	for _, chat := range chats {
		jid, err := types.ParseJID(chat.ID)
		if err != nil || !strings.Contains(jid.String(), "@g.us") {
			continue
		}
		if jid == v.Info.Chat {
			continue // leave this one last
		}
		if err := wac.LeaveGroup(ctx, jid); err != nil {
			failed++
		} else {
			left++
		}
		time.Sleep(1 * time.Second) // rate limit
	}

	// now leave current
	_ = ReplyErr(v, fmt.Sprintf("✅ Left *%d* groups (%d failed). Leaving this group too…", left, failed))
	time.Sleep(500 * time.Millisecond)
	_ = wac.LeaveGroup(ctx, v.Info.Chat)

	return nil
}

func joinGroupHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .joingroup <invite link>\nExample: .joingroup https://chat.whatsapp.com/ABC123")
	}

	link := args[0]
	// extract code from full link
	code := link
	if idx := strings.LastIndex(link, "/"); idx != -1 {
		code = link[idx+1:]
	}
	// strip query string
	if idx := strings.Index(code, "?"); idx != -1 {
		code = code[:idx]
	}

	if code == "" {
		return ReplyErr(v, "❌ Invalid invite link.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	groupJID, err := wac.JoinGroupWithLink(ctx, code)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to join group: %v", err))
	}

	return ReplyErr(v, fmt.Sprintf("✅ Joined group: `%s`", groupJID.String()))
}

func init() {
	Register(ModuleInfo{
		Name:        "Leave/Join Group",
		Description: "Leave current group, leave all groups, or join via invite link",
		Commands: []CommandInfo{
			{Pattern: "leave", Handler: leaveGroupHandler, Sudo: true},
			{Pattern: "leaveall", Handler: leaveAllGroupsHandler, Sudo: true},
			{Pattern: "joingroup", Handler: joinGroupHandler, Sudo: true},
		},
	})
}
