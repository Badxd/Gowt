package modules

import (
	"fmt"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// .adminonly on/off   — toggle group announce mode (only admins can send)
// .lockgroup          — only admins can edit group info
// .unlockgroup        — all members can edit group info
// .addmember <num>    — add a participant
// .removemember / .kick @user — remove a participant
// .promote @user      — promote to admin
// .demote @user       — demote from admin

var (
	adminOnlyMu    sync.RWMutex
	adminOnlyChats = make(map[string]bool) // chatJID -> true if announce mode tracked locally
)

func adminOnlyHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	chat := v.Info.Chat.String()

	if len(args) == 0 {
		adminOnlyMu.RLock()
		enabled := adminOnlyChats[chat]
		adminOnlyMu.RUnlock()

		status := "disabled 🔴"
		if enabled {
			status = "enabled 🟢"
		}
		return ReplyErr(v, fmt.Sprintf("👑 *Admin-only mode* is currently *%s*.", status))
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	ctx, cancel := newContext(15)
	defer cancel()

	switch strings.ToLower(args[0]) {
	case "on", "enable":
		err := wac.SetGroupAnnounce(ctx, v.Info.Chat, true)
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
		}
		adminOnlyMu.Lock()
		adminOnlyChats[chat] = true
		adminOnlyMu.Unlock()
		return ReplyErr(v, "👑 *Admin-only mode* enabled. Only admins can send messages.")

	case "off", "disable":
		err := wac.SetGroupAnnounce(ctx, v.Info.Chat, false)
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
		}
		adminOnlyMu.Lock()
		delete(adminOnlyChats, chat)
		adminOnlyMu.Unlock()
		return ReplyErr(v, "✅ *Admin-only mode* disabled. All members can send messages.")

	default:
		return ReplyErr(v, "Usage: .adminonly on/off")
	}
}

func lockGroupHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := newContext(15)
	defer cancel()
	if err := wac.SetGroupLocked(ctx, v.Info.Chat, true); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, "🔒 *Group locked.* Only admins can edit group info.")
}

func unlockGroupHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := newContext(15)
	defer cancel()
	if err := wac.SetGroupLocked(ctx, v.Info.Chat, false); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, "🔓 *Group unlocked.* All members can edit group info.")
}


func removeMemberHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .removemember @mention or .removemember <number>")
	}
	targetJID, parseErr := types.ParseJID(target)
	if parseErr != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Invalid JID: %v", parseErr))
	}
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := newContext(15)
	defer cancel()
	_, err2 := wac.UpdateGroupParticipants(ctx, v.Info.Chat, []types.JID{targetJID}, "remove")
	if err2 != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to remove member: %v", err2))
	}
	return ReplyErr(v, fmt.Sprintf("✅ *+%s* removed from the group.", targetJID.User))
}



func init() {
	Register(ModuleInfo{
		Name:        "Admin Controls",
		Description: "Admin-only mode, group lock/unlock, add/remove/promote/demote members",
		Commands: []CommandInfo{
			{Pattern: "adminonly", Handler: adminOnlyHandler, Sudo: true},
			{Pattern: "lockgroup", Handler: lockGroupHandler, Sudo: true},
			{Pattern: "unlockgroup", Handler: unlockGroupHandler, Sudo: true},
			{Pattern: "removemember", Handler: removeMemberHandler, Sudo: true},
		},
	})
}
