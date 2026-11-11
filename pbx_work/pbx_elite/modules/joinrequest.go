package modules

// Join Request module — manage group join requests when approval mode is on.
// Uses whatsmeow GetGroupRequestParticipants / RejectGroupParticipants.
//
// Commands:
//   .joinreq list            — list pending join requests in this group
//   .joinreq approve @user   — approve a specific join request
//   .joinreq reject @user    — reject a specific join request
//   .joinreq approveall      — approve all pending requests
//   .joinreq rejectall       — reject all pending requests
//
// Note: .joinapproval on/off (toggle approval mode) already exists in grouplink.go

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"PbxGb/client"
)

func joinReqHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}

	wac := client.Get()
	ctx := context.Background()
	groupJID := v.Info.Chat

	if len(args) == 0 {
		return ReplyErr(v, "Usage: .joinreq list / approve @user / reject @user / approveall / rejectall")
	}

	sub := strings.ToLower(args[0])

	// helper: fetch pending requests
	fetchPending := func() ([]types.GroupRequestParticipant, error) {
		return wac.GetGroupRequestParticipants(ctx, groupJID)
	}

	switch sub {
	case "list":
		pending, err := fetchPending()
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed to fetch requests: %v", err))
		}
		if len(pending) == 0 {
			return ReplyErr(v, "ℹ️ No pending join requests.")
		}
		lines := []string{fmt.Sprintf("📋 *%d pending join request(s):*\n", len(pending))}
		for i, p := range pending {
			lines = append(lines, fmt.Sprintf("%d. +%s", i+1, p.JID.User))
		}
		return ReplyErr(v, strings.Join(lines, "\n"))

	case "approve":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .joinreq approve @user")
		}
		target, err := extractTargetJID(v, args[1:])
		if err != nil || target == "" {
			return ReplyErr(v, "❌ Mention a user or provide a number.")
		}
		targetJID, err := types.ParseJID(target)
		if err != nil {
			return ReplyErr(v, "❌ Invalid JID: "+target)
		}
		_, err = wac.UpdateGroupParticipants(ctx, groupJID, []types.JID{targetJID}, "approve")
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Approve failed: %v", err))
		}
		return ReplyErr(v, fmt.Sprintf("✅ Approved: *+%s*", targetJID.User))

	case "reject":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .joinreq reject @user")
		}
		target, err := extractTargetJID(v, args[1:])
		if err != nil || target == "" {
			return ReplyErr(v, "❌ Mention a user or provide a number.")
		}
		targetJID, err := types.ParseJID(target)
		if err != nil {
			return ReplyErr(v, "❌ Invalid JID: "+target)
		}
		_, err = wac.RejectGroupParticipants(ctx, groupJID, []types.JID{targetJID})
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Reject failed: %v", err))
		}
		return ReplyErr(v, fmt.Sprintf("🚫 Rejected: *+%s*", targetJID.User))

	case "approveall":
		pending, err := fetchPending()
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed to fetch requests: %v", err))
		}
		if len(pending) == 0 {
			return ReplyErr(v, "ℹ️ No pending join requests.")
		}
		jids := make([]types.JID, len(pending))
		for i, p := range pending {
			jids[i] = p.JID
		}
		_, err = wac.UpdateGroupParticipants(ctx, groupJID, jids, "approve")
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Approve all failed: %v", err))
		}
		return ReplyErr(v, fmt.Sprintf("✅ Approved *%d* pending join request(s).", len(jids)))

	case "rejectall":
		pending, err := fetchPending()
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed to fetch requests: %v", err))
		}
		if len(pending) == 0 {
			return ReplyErr(v, "ℹ️ No pending join requests.")
		}
		jids := make([]types.JID, len(pending))
		for i, p := range pending {
			jids[i] = p.JID
		}
		_, err = wac.RejectGroupParticipants(ctx, groupJID, jids)
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Reject all failed: %v", err))
		}
		return ReplyErr(v, fmt.Sprintf("🚫 Rejected *%d* pending join request(s).", len(jids)))

	default:
		return ReplyErr(v, "❌ Unknown sub-command. Use: list / approve / reject / approveall / rejectall")
	}
}

func init() {
	Register(ModuleInfo{
		Name:        "JoinRequest",
		Description: "Manage group join requests (list, approve, reject) when approval mode is on",
		Commands: []CommandInfo{
			{Pattern: "joinreq",   Handler: joinReqHandler, Sudo: true},
			{Pattern: "joinqueue", Handler: joinReqHandler, Sudo: true},
		},
	})
}
