package modules

import (
	"context"
	"fmt"
	"strings"
	"time"


	"go.mau.fi/whatsmeow/types/events"
)

// .groupinfo   — show detailed info about the current group
// .invitelink  — get group invite link
// .setname <name> — set group name

func groupInfoDetailHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
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

	adminCount := 0
	for _, p := range info.Participants {
		if p.IsAdmin || p.IsSuperAdmin {
			adminCount++
		}
	}

	lines := []string{
		fmt.Sprintf("👥 *Group Info*"),
		"",
		fmt.Sprintf("📌 Name: *%s*", info.GroupName.Name),
	}

	if info.Topic.Topic != "" {
		desc := info.Topic.Topic
		if len(desc) > 100 {
			desc = desc[:100] + "…"
		}
		lines = append(lines, fmt.Sprintf("📝 Description: _%s_", desc))
	}

	lines = append(lines,
		fmt.Sprintf("👤 Members: *%d* (%d admins)", len(info.Participants), adminCount),
		fmt.Sprintf("📅 Created: %s", info.GroupCreated.Format("02 Jan 2006")),
		fmt.Sprintf("🆔 JID: `%s`", info.JID.String()),
	)

	if info.IsAnnounce {
		lines = append(lines, "🔒 Mode: *Announcement only*")
	}
	if info.IsLocked {
		lines = append(lines, "🔒 Settings: *Locked* (admin-only edits)")
	}

	// owner
	ownerPhone := ""
	if !info.OwnerJID.IsEmpty() {
		ownerPhone = "+" + info.OwnerJID.User
	}
	if ownerPhone != "" {
		lines = append(lines, fmt.Sprintf("👑 Owner: %s", ownerPhone))
	}

	return ReplyErr(v, strings.Join(lines, "\n"))
}

func inviteLinkHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	link, err := wac.GetGroupInviteLink(ctx, v.Info.Chat, false)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to get invite link: %v", err))
	}

	return ReplyErr(v, fmt.Sprintf("🔗 *Group Invite Link:*\nhttps://chat.whatsapp.com/%s", link))
}

func revokeInviteLinkHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	newLink, err := wac.GetGroupInviteLink(ctx, v.Info.Chat, true) // reset=true
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to revoke link: %v", err))
	}

	return ReplyErr(v, fmt.Sprintf("✅ Invite link revoked.\n🔗 New link:\nhttps://chat.whatsapp.com/%s", newLink))
}


func listGroupMembersHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	ctx2, cancel2 := newContext(15)
	defer cancel2()
	info, err := wac.GetGroupInfo(ctx2, v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to get group info: %v", err))
	}

	var admins []string
	var members []string

	for _, p := range info.Participants {
		phone := "+" + p.JID.User
		if p.IsSuperAdmin {
			admins = append(admins, fmt.Sprintf("👑 %s (super admin)", phone))
		} else if p.IsAdmin {
			admins = append(admins, fmt.Sprintf("⭐ %s (admin)", phone))
		} else {
			members = append(members, fmt.Sprintf("👤 %s", phone))
		}
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("👥 *Members of %s* (%d total)", info.GroupName.Name, len(info.Participants)), "")

	if len(admins) > 0 {
		lines = append(lines, "*Admins:*")
		lines = append(lines, admins...)
		lines = append(lines, "")
	}

	if len(members) > 0 {
		// cap at 50 to avoid message too long
		shown := members
		extra := 0
		if len(members) > 50 {
			shown = members[:50]
			extra = len(members) - 50
		}
		lines = append(lines, "*Members:*")
		lines = append(lines, shown...)
		if extra > 0 {
			lines = append(lines, fmt.Sprintf("… and %d more", extra))
		}
	}

	return ReplyErr(v, strings.Join(lines, "\n"))
}

func init() {
	Register(ModuleInfo{
		Name:        "Group Info",
		Description: "Get group info, invite links, set group name, list members",
		Commands: []CommandInfo{
			{Pattern: "ginfo", Handler: groupInfoDetailHandler, Sudo: true},
			{Pattern: "invitelink", Handler: inviteLinkHandler, Sudo: true},
			{Pattern: "revokeinvite", Handler: revokeInviteLinkHandler, Sudo: true},
			{Pattern: "members", Handler: listGroupMembersHandler, Sudo: true},
		},
	})
}
