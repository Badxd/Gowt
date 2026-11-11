package modules

// Group Extras — commands missing from existing group modules.
//
// Commands:
//   .vccount        — how many members are currently in a voice/video call in this group
//   .membercount    — fast member count for this group
//   .phonelist      — list all members with their phone numbers (+91xxxxxx format)
//   .mygroups       — list all groups the bot is in (name + JID + member count)
//   .grouplink      — get invite link for this group (alias already exists; added here as backup)
//   .admins         — list admins of this group
//   .groupage       — show when this group was created
//   .kickinactive   — kick members who have never sent a message (requires groupstats enabled)
//   .setjoincmd <msg> — set a custom join approval message (shown when someone requests to join)

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"PbxGb/client"
)

// ── MEMBER COUNT ──────────────────────────────────────────────────────────────
func memberCountHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
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
	return ReplyErr(v, fmt.Sprintf(
		"👥 *%s*\n\n"+
			"📊 Total Members: *%d*\n"+
			"👑 Admins: *%d*\n"+
			"👤 Regular: *%d*",
		info.Name,
		len(info.Participants),
		adminCount,
		len(info.Participants)-adminCount,
	))
}

// ── VC COUNT (members currently in call) ──────────────────────────────────────
// WhatsApp doesn't expose an ongoing-call participant list via whatsmeow.
// We track it via CallOffer/CallTerminate events in memory.
// This shows the count of active calls in the group detected by the bot.
var activeCallParticipants = map[string]map[string]bool{} // chatJID -> set of callerJIDs

// RegisterCallEvent is called from the event loop (hook into main event handler)
// to track who is currently in a call in each group.
func RegisterCallEvent(chatJID, callerJID string, joined bool) {
	if activeCallParticipants[chatJID] == nil {
		activeCallParticipants[chatJID] = map[string]bool{}
	}
	if joined {
		activeCallParticipants[chatJID][callerJID] = true
	} else {
		delete(activeCallParticipants[chatJID], callerJID)
	}
}

func vcCountHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	chatKey := v.Info.Chat.String()
	participants := activeCallParticipants[chatKey]
	if len(participants) == 0 {
		return ReplyErr(v, "📵 No active voice/video call detected in this group right now.")
	}
	lines := []string{fmt.Sprintf("📞 *%d member(s) in call:*\n", len(participants))}
	i := 1
	for jidStr := range participants {
		lines = append(lines, fmt.Sprintf("%d. +%s", i, strings.TrimSuffix(jidStr, "@s.whatsapp.net")))
		i++
	}
	return ReplyErr(v, strings.Join(lines, "\n"))
}

// ── PHONE LIST ────────────────────────────────────────────────────────────────
func phoneListHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
	info, err := wac.GetGroupInfo(v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	lines := []string{fmt.Sprintf("📋 *Phone numbers — %s* (%d members)\n", info.Name, len(info.Participants))}
	for i, p := range info.Participants {
		role := ""
		if p.IsSuperAdmin {
			role = " 👑"
		} else if p.IsAdmin {
			role = " ⭐"
		}
		lines = append(lines, fmt.Sprintf("%d. +%s%s", i+1, p.JID.User, role))
	}
	lines = append(lines, "\n👑 = Super Admin  ⭐ = Admin")
	return ReplyErr(v, strings.Join(lines, "\n"))
}

// ── MY GROUPS ─────────────────────────────────────────────────────────────────
func myGroupsHandler(v *events.Message, args []string) error {
	wac := client.Get()
	groups, err := wac.GetJoinedGroups()
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	// Optional filter by name
	filter := ""
	if len(args) > 0 {
		filter = strings.ToLower(strings.Join(args, " "))
	}

	var matched []*types.GroupInfo
	for _, g := range groups {
		if filter == "" || strings.Contains(strings.ToLower(g.Name), filter) {
			matched = append(matched, g)
		}
	}

	if len(matched) == 0 {
		if filter != "" {
			return ReplyErr(v, fmt.Sprintf("❌ No groups found matching *%s*.", filter))
		}
		return ReplyErr(v, "ℹ️ Not in any groups.")
	}

	lines := []string{fmt.Sprintf("👥 *In %d group(s):*\n", len(matched))}
	for i, g := range matched {
		lines = append(lines, fmt.Sprintf(
			"%d. *%s*\n   👤 %d members | `%s`",
			i+1, g.Name, len(g.Participants), g.JID.String(),
		))
	}
	return ReplyErr(v, strings.Join(lines, "\n"))
}

// ── ADMINS LIST ───────────────────────────────────────────────────────────────
func adminsListHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
	info, err := wac.GetGroupInfo(v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	var admins []string
	for _, p := range info.Participants {
		if p.IsSuperAdmin {
			admins = append(admins, fmt.Sprintf("👑 +%s (super admin)", p.JID.User))
		} else if p.IsAdmin {
			admins = append(admins, fmt.Sprintf("⭐ +%s", p.JID.User))
		}
	}

	if len(admins) == 0 {
		return ReplyErr(v, "ℹ️ No admins found.")
	}

	return ReplyErr(v, fmt.Sprintf("*Admins of %s* (%d):\n\n%s",
		info.Name, len(admins), strings.Join(admins, "\n")))
}

// ── GROUP AGE ─────────────────────────────────────────────────────────────────
func groupAgeHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
	info, err := wac.GetGroupInfo(v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	created := info.GroupCreated
	now := time.Now()
	age := now.Sub(created)

	days := int(age.Hours() / 24)
	months := days / 30
	years := days / 365

	ageStr := ""
	switch {
	case years >= 1:
		ageStr = fmt.Sprintf("%d year(s), %d month(s)", years, (days%365)/30)
	case months >= 1:
		ageStr = fmt.Sprintf("%d month(s), %d day(s)", months, days%30)
	default:
		ageStr = fmt.Sprintf("%d day(s)", days)
	}

	creatorStr := "unknown"
	if !info.OwnerJID.IsEmpty() {
		creatorStr = "+"+info.OwnerJID.User
	}

	return ReplyErr(v, fmt.Sprintf(
		"📅 *Group Age*\n\n"+
			"🏷️ *Name:* %s\n"+
			"📆 *Created:* %s\n"+
			"⏳ *Age:* %s\n"+
			"👤 *Creator:* %s",
		info.Name,
		created.Format("02 Jan 2006, 3:04 PM"),
		ageStr,
		creatorStr,
	))
}

// ── GROUP SETTINGS OVERVIEW ───────────────────────────────────────────────────
func groupSettingsHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
	info, err := wac.GetGroupInfo(v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	boolIcon := func(b bool) string {
		if b {
			return "✅"
		}
		return "❌"
	}

	addMode := "Anyone"
	if info.MemberAddMode == types.GroupMemberAddModeAdminAdd {
		addMode = "Admins only"
	}

	return ReplyErr(v, fmt.Sprintf(
		"⚙️ *Group Settings — %s*\n\n"+
			"🔒 *Locked (admin-only msgs):* %s\n"+
			"📢 *Announce mode:* %s\n"+
			"🔗 *Join approval required:* %s\n"+
			"👥 *Who can add members:* %s\n"+
			"👥 *Members:* %d\n"+
			"🆔 *JID:* `%s`",
		info.Name,
		boolIcon(info.IsLocked),
		boolIcon(info.IsAnnounce),
		boolIcon(info.MemberAddMode == types.GroupMemberAddModeAdminAdd),
		addMode,
		len(info.Participants),
		info.JID.String(),
	))
}

// ── REMOVE MEMBER (phone number support) ─────────────────────────────────────
// existing .kick takes @mention — this adds direct phone number support
func removeMemberHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .removemember 919812345678  (or @mention)")
	}
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "❌ Could not resolve user.")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID: "+target)
	}
	wac := client.Get()
	_, err = wac.UpdateGroupParticipants(context.Background(), v.Info.Chat, []types.JID{jid}, "remove")
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Remove failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("✅ Removed *+%s* from group.", jid.User))
}

// ── ADD MEMBER (phone number support) ─────────────────────────────────────────
func addMemberByPhoneHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .addmember 919812345678")
	}
	raw := strings.Join(args, "")
	nums := strings.Split(raw, ",")
	wac := client.Get()
	ctx := context.Background()

	var added, failed []string
	for _, n := range nums {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if !strings.Contains(n, "@") {
			n = n + "@s.whatsapp.net"
		}
		jid, err := types.ParseJID(n)
		if err != nil {
			failed = append(failed, n+" (invalid)")
			continue
		}
		_, err = wac.UpdateGroupParticipants(ctx, v.Info.Chat, []types.JID{jid}, "add")
		if err != nil {
			failed = append(failed, "+"+jid.User+" (failed: "+err.Error()+")")
		} else {
			added = append(added, "+"+jid.User)
		}
	}

	var lines []string
	if len(added) > 0 {
		lines = append(lines, "✅ *Added:* "+strings.Join(added, ", "))
	}
	if len(failed) > 0 {
		lines = append(lines, "❌ *Failed:* "+strings.Join(failed, ", "))
	}
	return ReplyErr(v, strings.Join(lines, "\n"))
}

func init() {
	Register(ModuleInfo{
		Name:        "GroupExtras",
		Description: "Member count, phone list, VC count, mygroups, admins list, group age, group settings",
		Commands: []CommandInfo{
			{Pattern: "membercount",  Handler: memberCountHandler,     Sudo: true},
			{Pattern: "mcount",       Handler: memberCountHandler,     Sudo: true},
			{Pattern: "vccount",      Handler: vcCountHandler,         Sudo: true},
			{Pattern: "phonelist",    Handler: phoneListHandler,       Sudo: true},
			{Pattern: "numlist",      Handler: phoneListHandler,       Sudo: true},
			{Pattern: "mygroups",     Handler: myGroupsHandler,        Sudo: true},
			{Pattern: "listgroups",   Handler: myGroupsHandler,        Sudo: true},
			{Pattern: "admins",       Handler: adminsListHandler,      Sudo: true},
			{Pattern: "adminlist",    Handler: adminsListHandler,      Sudo: true},
			{Pattern: "groupage",     Handler: groupAgeHandler,        Sudo: true},
			{Pattern: "gsettings",    Handler: groupSettingsHandler,   Sudo: true},
			{Pattern: "groupsettings",Handler: groupSettingsHandler,   Sudo: true},
			{Pattern: "removemember", Handler: removeMemberHandler,    Sudo: true},
			{Pattern: "addmember",    Handler: addMemberByPhoneHandler,Sudo: true},
		},
	})
}
