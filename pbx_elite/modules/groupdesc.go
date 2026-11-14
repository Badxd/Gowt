package modules

import (
	"context"
	"fmt"
	"strings"
	"time"


	"go.mau.fi/whatsmeow/types/events"
)

// .setdesc <text>   — set group description
// .getdesc          — show current group description

func setGroupDescHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ Groups only.")
	}

	if len(args) == 0 {
		return ReplyErr(v, "Usage: .setdesc <new description>")
	}

	desc := strings.Join(args, " ")

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := wac.SetGroupDescription(ctx, v.Info.Chat, desc)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to set description: %v", err))
	}

	preview := desc
	if len(preview) > 60 {
		preview = preview[:60] + "..."
	}
	return ReplyErr(v, fmt.Sprintf("✅ Group description updated:\n_%s_", preview))
}

func getGroupDescHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ Groups only.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	info, err := wac.GetGroupInfo(ctx, v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to get group info: %v", err))
	}

	desc := info.Desc
	if desc == "" {
		return ReplyErr(v, "ℹ️ This group has no description set.")
	}

	return ReplyErr(v, fmt.Sprintf("📝 *Group Description:*\n\n%s", desc))
}

func init() {
	Register(ModuleInfo{
		Name:        "Group Description",
		Description: "Get or set group description",
		Commands: []CommandInfo{
			{Pattern: "setdesc", Handler: setGroupDescHandler, Sudo: true},
			{Pattern: "getdesc", Handler: getGroupDescHandler, Sudo: true},
		},
	})
}
