package modules

// Group Description & Link Tools — group description, subject, invite links.
//
// Commands:
//   .setdesc <text>       — set group description
//   .getdesc              — show group description
//   .setsubject <name>    — rename group
//   .grouplink            — get invite link for this group
//   .revokelink           — revoke and regenerate invite link
//   .groupjoin <link>     — join a group via invite link
//   .groupleave           — leave the current group
//   .setgrouppp           — set group photo (reply to image)
//   .delgrouppp           — remove group photo
//   .groupreport          — full group report (members, admins, settings, link)

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"PbxGb/client"
)

// ── SET DESCRIPTION ───────────────────────────────────────────────────────────
func setGroupDescHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
	ctx := context.Background()
	desc := strings.Join(args, " ")
	if err := wac.SetGroupDescription(ctx, v.Info.Chat, desc); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	if desc == "" {
		return ReplyErr(v, "✅ Group description cleared.")
	}
	return ReplyErr(v, "✅ Group description updated.")
}

// ── GET DESCRIPTION ───────────────────────────────────────────────────────────
func getGroupDescHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
	info, err := wac.GetGroupInfo(v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	if info.Desc == "" {
		return ReplyErr(v, "ℹ️ This group has no description.")
	}
	return ReplyErr(v, fmt.Sprintf("📝 *Group Description:*\n\n%s", info.Desc))
}

// ── SET SUBJECT / RENAME ──────────────────────────────────────────────────────
func setGroupSubjectHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .setsubject <new group name>")
	}
	name := strings.Join(args, " ")
	wac := client.Get()
	ctx := context.Background()
	if err := wac.SetGroupName(ctx, v.Info.Chat, name); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("✅ Group renamed to *%s*.", name))
}

// ── GET INVITE LINK ───────────────────────────────────────────────────────────
func groupLinkHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
	ctx := context.Background()
	link, err := wac.GetGroupInviteLink(ctx, v.Info.Chat, false)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("🔗 *Group Invite Link:*\nhttps://chat.whatsapp.com/%s", link))
}

// ── REVOKE INVITE LINK ────────────────────────────────────────────────────────
func revokeLinkHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
	ctx := context.Background()
	link, err := wac.GetGroupInviteLink(ctx, v.Info.Chat, true) // true = revoke
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("🔄 *New invite link (old one revoked):*\nhttps://chat.whatsapp.com/%s", link))
}

// ── JOIN GROUP VIA LINK ───────────────────────────────────────────────────────
func groupJoinHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .groupjoin <invite_link_or_code>")
	}
	code := args[0]
	// strip full URL if pasted
	code = strings.TrimPrefix(code, "https://chat.whatsapp.com/")
	code = strings.TrimPrefix(code, "http://chat.whatsapp.com/")
	code = strings.TrimSpace(code)

	wac := client.Get()
	ctx := context.Background()
	groupInfo, err := wac.JoinGroupWithLink(ctx, code)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Join failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf(
		"✅ Joined group *%s*\nJID: `%s`\nMembers: %d",
		groupInfo.Name, groupInfo.JID.String(), len(groupInfo.Participants),
	))
}

// ── LEAVE GROUP ───────────────────────────────────────────────────────────────
func groupLeaveHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
	ctx := context.Background()

	// Get name first for the confirmation message
	info, _ := wac.GetGroupInfo(v.Info.Chat)
	name := v.Info.Chat.User
	if info != nil {
		name = info.Name
	}

	if err := wac.LeaveGroup(ctx, v.Info.Chat); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to leave: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("👋 Left group *%s*.", name))
}

// ── SET GROUP PHOTO ───────────────────────────────────────────────────────────
func setGroupPPHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	img := getQuotedImageBytes(v)
	if img == nil {
		return ReplyErr(v, "❌ Reply to an image to set it as the group photo.")
	}
	wac := client.Get()
	ctx := context.Background()
	_, err := wac.SetGroupPhoto(ctx, v.Info.Chat, img)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, "✅ Group photo updated.")
}

// ── DELETE GROUP PHOTO ────────────────────────────────────────────────────────
func delGroupPPHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
	ctx := context.Background()
	_, err := wac.SetGroupPhoto(ctx, v.Info.Chat, nil)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, "✅ Group photo removed.")
}

// ── GROUP REPORT ──────────────────────────────────────────────────────────────
func groupReportHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
	ctx := context.Background()

	info, err := wac.GetGroupInfo(v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	// Count roles
	admins, superAdmins, regular := 0, 0, 0
	for _, p := range info.Participants {
		switch {
		case p.IsSuperAdmin:
			superAdmins++
		case p.IsAdmin:
			admins++
		default:
			regular++
		}
	}

	// Get invite link
	link, linkErr := wac.GetGroupInviteLink(ctx, v.Info.Chat, false)
	linkLine := "_(not available)_"
	if linkErr == nil {
		linkLine = "https://chat.whatsapp.com/" + link
	}

	desc := info.Desc
	if desc == "" {
		desc = "_(no description)_"
	}
	if len(desc) > 200 {
		desc = desc[:200] + "…"
	}

	boolIcon := func(b bool) string {
		if b {
			return "✅"
		}
		return "❌"
	}

	return ReplyErr(v, fmt.Sprintf(
		"📊 *Group Report*\n\n"+
			"🏷️ *Name:* %s\n"+
			"🆔 *JID:* `%s`\n"+
			"📅 *Created:* %s\n"+
			"👤 *Creator:* +%s\n\n"+
			"👥 *Members:* %d total\n"+
			"  👑 Super admins: %d\n"+
			"  ⭐ Admins: %d\n"+
			"  👤 Regular: %d\n\n"+
			"📝 *Description:*\n%s\n\n"+
			"⚙️ *Settings:*\n"+
			"  🔒 Locked: %s\n"+
			"  📢 Announce: %s\n\n"+
			"🔗 *Invite link:*\n%s",
		info.Name,
		info.JID.String(),
		info.GroupCreated.Format("02 Jan 2006"),
		info.OwnerJID.User,
		len(info.Participants),
		superAdmins, admins, regular,
		desc,
		boolIcon(info.IsLocked),
		boolIcon(info.IsAnnounce),
		linkLine,
	))
}

// ── PROMOTE / DEMOTE ─────────────────────────────────────────────────────────
func promoteHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .promote @mention or .promote 919xxx")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}
	wac := client.Get()
	_, err = wac.UpdateGroupParticipants(context.Background(), v.Info.Chat, []types.JID{jid}, "promote")
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("✅ Promoted *+%s* to admin.", jid.User))
}

func demoteHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .demote @mention or .demote 919xxx")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}
	wac := client.Get()
	_, err = wac.UpdateGroupParticipants(context.Background(), v.Info.Chat, []types.JID{jid}, "demote")
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("✅ Demoted *+%s* from admin.", jid.User))
}

func init() {
	Register(ModuleInfo{
		Name:        "GroupDesc",
		Description: "Group description, rename, invite link, join/leave, group photo, promote/demote, full report",
		Commands: []CommandInfo{
			{Pattern: "setdesc",    Handler: setGroupDescHandler,    Sudo: true},
			{Pattern: "getdesc",    Handler: getGroupDescHandler,    Sudo: true},
			{Pattern: "setsubject", Handler: setGroupSubjectHandler, Sudo: true},
			{Pattern: "rename",     Handler: setGroupSubjectHandler, Sudo: true},
			{Pattern: "grouplink",  Handler: groupLinkHandler,       Sudo: true},
			{Pattern: "invitelink", Handler: groupLinkHandler,       Sudo: true},
			{Pattern: "revokelink", Handler: revokeLinkHandler,      Sudo: true},
			{Pattern: "groupjoin",  Handler: groupJoinHandler,       Sudo: true},
			{Pattern: "joingroup",  Handler: groupJoinHandler,       Sudo: true},
			{Pattern: "groupleave", Handler: groupLeaveHandler,      Sudo: true},
			{Pattern: "leavegroup", Handler: groupLeaveHandler,      Sudo: true},
			{Pattern: "setgrouppp", Handler: setGroupPPHandler,      Sudo: true},
			{Pattern: "delgrouppp", Handler: delGroupPPHandler,      Sudo: true},
			{Pattern: "groupreport",Handler: groupReportHandler,     Sudo: true},
			{Pattern: "greport",    Handler: groupReportHandler,     Sudo: true},
			{Pattern: "promote",    Handler: promoteHandler,         Sudo: true},
			{Pattern: "demote",     Handler: demoteHandler,          Sudo: true},
		},
	})
}
