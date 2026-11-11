package modules

import (
	"context"
	"fmt"
	"strings"
	"time"


	"go.mau.fi/whatsmeow/types/events"
)

// Group description / subject commands
// .setdesc <text>  — set group description
// .getdesc         — show current group description
// .setname <text>  — set group subject/name  (alias setgname already exists, but use setname here)
// .getname         — show current group name
// .setpic          — set group icon (reply to image)

func setDescHandler(v *events.Message, args []string) error {
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

	if err := wac.SetGroupDescription(ctx, v.Info.Chat, "", desc); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, "✅ Group description updated.")
}

func getDescHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ Groups only.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	info, err := wac.GetGroupInfo(v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	desc := info.Topic
	if desc == "" {
		desc = "(no description set)"
	}
	return ReplyErr(v, fmt.Sprintf("📋 *Group Description:*\n%s", desc))
}

func getNameHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ Groups only.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	info, err := wac.GetGroupInfo(v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("👥 *Group Name:* %s", info.Name))
}

func setGroupPicHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ Groups only.")
	}

	_, _, err := downloadImageFromMsg(v)
	if err != nil {
		return ReplyErr(v, "❌ Reply to an image with .setpic")
	}

	// whatsmeow SetGroupPhoto takes the JPEG bytes
	imgData, mime, err := downloadImageFromMsg(v)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Could not download image: %v", err))
	}
	_ = mime

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_, err = wac.SetGroupPhoto(ctx, v.Info.Chat, imgData)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to set group photo: %v", err))
	}
	return ReplyErr(v, "✅ Group photo updated.")
}

func groupInfoChatHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ Groups only.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	info, err := wac.GetGroupInfo(v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	adminCount := 0
	for _, p := range info.Participants {
		if p.IsAdmin || p.IsSuperAdmin {
			adminCount++
		}
	}

	created := info.GroupCreated.Format("02 Jan 2006")
	desc := info.Topic
	if desc == "" {
		desc = "(none)"
	}

	return ReplyErr(v, fmt.Sprintf(
		"👥 *Group Info*\n"+
			"📌 *Name:* %s\n"+
			"🆔 *JID:* %s\n"+
			"👤 *Members:* %d\n"+
			"🛡️ *Admins:* %d\n"+
			"📅 *Created:* %s\n"+
			"📋 *Desc:* %s",
		info.Name,
		info.JID.String(),
		len(info.Participants),
		adminCount,
		created,
		desc,
	))
}

func init() {
	Register(ModuleInfo{
		Name:        "Chat Tools",
		Description: "Group description, name, photo, and info commands",
		Commands: []CommandInfo{
			{Pattern: "getname", Handler: getNameHandler, Sudo: true},
			{Pattern: "setpic",    Handler: setGroupPicHandler, Sudo: true},
			{Pattern: "chatdesc",  Handler: getDescHandler,     Sudo: false},
			{Pattern: "setchatdesc",Handler: setDescHandler,    Sudo: true},
		},
	})
}
