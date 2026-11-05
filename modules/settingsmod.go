package modules

import (
	"context"
	"fmt"
	"strings"
	"time"


	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// .lock    — lock group (only admins can send messages)
// .unlock  — unlock group (everyone can send messages)
// .gcopen  — set group to allow all participants to change group settings
// .gcclose — set group to only admins can change group settings



func gcOpenHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ Groups only.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// SetGroupLocked(false) = all participants can edit group info
	err := wac.SetGroupLocked(ctx, v.Info.Chat, false)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	return ReplyErr(v, "✅ Group settings unlocked — all participants can edit group info.")
}

func gcCloseHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ Groups only.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// SetGroupLocked(true) = only admins can edit group info
	err := wac.SetGroupLocked(ctx, v.Info.Chat, true)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	return ReplyErr(v, "✅ Group settings locked — only admins can edit group info.")
}

func groupJoinApprovalHandler(v *events.Message, args []string) error {
	// Toggle join approval (membership approval required)
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ Groups only.")
	}

	if len(args) == 0 {
		return ReplyErr(v, "Usage: .joinapproval on/off")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var enable bool
	switch strings.ToLower(args[0]) {
	case "on", "true", "1":
		enable = true
	case "off", "false", "0":
		enable = false
	default:
		return ReplyErr(v, "Usage: .joinapproval on/off")
	}

	err := wac.SetGroupMemberAddMode(ctx, v.Info.Chat, whatsmeow.GroupMemberAddModeAdminAdd)
	_ = enable
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	if enable {
		return ReplyErr(v, "✅ Join approval enabled — admin must approve new members.")
	}
	return ReplyErr(v, "✅ Join approval disabled.")
}

func setGroupNameHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ Groups only.")
	}

	if len(args) == 0 {
		return ReplyErr(v, "Usage: .setname <new group name>")
	}

	name := strings.Join(args, " ")

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := wac.SetGroupName(ctx, v.Info.Chat, name)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to set name: %v", err))
	}

	return ReplyErr(v, fmt.Sprintf("✅ Group name changed to: *%s*", name))
}

func init() {
	Register(ModuleInfo{
		Name:        "Group Settings",
		Description: "Lock/unlock group, manage group settings",
		Commands: []CommandInfo{
			{Pattern: "lock", Handler: lockGroupHandler, Sudo: true},
			{Pattern: "unlock", Handler: unlockGroupHandler, Sudo: true},
			{Pattern: "gcopen", Handler: gcOpenHandler, Sudo: true},
			{Pattern: "gcclose", Handler: gcCloseHandler, Sudo: true},
			{Pattern: "joinapproval", Handler: groupJoinApprovalHandler, Sudo: true},
			{Pattern: "setname", Handler: setGroupNameHandler, Sudo: true},
		},
	})
}
