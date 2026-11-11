package modules

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"PbxGb/client"
	"PbxGb/database"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// isGroup — check if message is from a group
func isGroup(v *events.Message) bool {
	return v.Info.Chat.Server == types.GroupServer
}

// ── GMUTE / GUNMUTE (group mute) ─────────────────────────────────────────────
// .gmute — sirf admins bolne de sakte ne
func gmuteHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
	if err := wac.SetGroupAnnounce(context.Background(), v.Info.Chat, true); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Gmute failed: %v", err))
	}
	return ReplyErr(v, "🔇 *Group Muted!*\nSirf admins message kar sakte ne.\nUnmute: `.gunmute`")
}

// .gunmute — sab bolne de sakte ne
func gunmuteHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
	if err := wac.SetGroupAnnounce(context.Background(), v.Info.Chat, false); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Gunmute failed: %v", err))
	}
	return ReplyErr(v, "🔊 *Group Unmuted!*\nSab members message kar sakte ne.")
}

// ── GBAN (group ban = kick + block) ──────────────────────────────────────────
var gbannedMu sync.RWMutex
var gbanned   = make(map[string]map[string]bool) // groupJID -> userJID -> banned

// .gban — group se kick karo + wapas aane se roko
func gbanHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}

	target := resolveTarget(v, args)
	if target == "" {
		return ReplyErr(v, "❌ Reply to user or: .gban 9198xxxxxx")
	}

	targetJID, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID: "+target)
	}

	wac := client.Get()
	groupStr := v.Info.Chat.String()

	// Kick from group
	_, err = wac.UpdateGroupParticipants(
		context.Background(),
		v.Info.Chat,
		[]types.JID{targetJID},
		"remove",
	)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Gban kick failed: %v", err))
	}

	// Add to gban list
	gbannedMu.Lock()
	if gbanned[groupStr] == nil {
		gbanned[groupStr] = make(map[string]bool)
	}
	gbanned[groupStr][target] = true
	gbannedMu.Unlock()

	_ = database.SaveGBanned(groupStr, target)

	user := strings.Split(target, "@")[0]
	return ReplyErr(v, fmt.Sprintf(
		"🚫 *GBanned!*\n\n👤 User: *+%s*\n📌 Group: %s\n\nUser kicked & banned from rejoining.",
		user, v.Info.Chat.String(),
	))
}

// .ungban — gban hatao
func ungbanHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}

	target := resolveTarget(v, args)
	if target == "" {
		return ReplyErr(v, "❌ Reply to user or: .ungban 9198xxxxxx")
	}

	groupStr := v.Info.Chat.String()
	gbannedMu.Lock()
	if gbanned[groupStr] != nil {
		delete(gbanned[groupStr], target)
	}
	gbannedMu.Unlock()

	_ = database.DeleteGBanned(groupStr, target)

	user := strings.Split(target, "@")[0]
	return ReplyErr(v, fmt.Sprintf("✅ GBan removed: *+%s*\nUser can rejoin now.", user))
}

// .gbanlist — gban list dekho
func gbanListHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}

	groupStr := v.Info.Chat.String()
	gbannedMu.RLock()
	list := gbanned[groupStr]
	gbannedMu.RUnlock()

	if len(list) == 0 {
		return ReplyErr(v, "ℹ️ No gbanned users in this group.")
	}

	lines := []string{"╔══════════════════════╗", "║   🚫 GBAN LIST        ║", "╚══════════════════════╝", ""}
	i := 1
	for jid := range list {
		user := strings.Split(jid, "@")[0]
		lines = append(lines, fmt.Sprintf("%d. +%s", i, user))
		i++
	}
	return ReplyErr(v, strings.Join(lines, "\n"))
}

// HandleGbanRejoin — jab koi group join kare, check karo gban list
func HandleGbanRejoin(v *events.Message) {
	if !isGroup(v) {
		return
	}
	groupStr := v.Info.Chat.String()
	sender := v.Info.Sender.String()

	gbannedMu.RLock()
	isBanned := gbanned[groupStr] != nil && gbanned[groupStr][sender]
	gbannedMu.RUnlock()

	if !isBanned {
		return
	}

	// Wapas kick karo
	wac := client.Get()
	_, _ = wac.UpdateGroupParticipants(
		context.Background(),
		v.Info.Chat,
		[]types.JID{v.Info.Sender},
		"remove",
	)
}

// ── TAGALL ────────────────────────────────────────────────────────────────────
func tagAllGroupHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ This command only works in groups.")
	}

	wac := client.Get()
	info, err := wac.GetGroupInfo(context.Background(), v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to get group info: %v", err))
	}

	extra := ""
	if len(args) > 0 {
		extra = strings.Join(args, " ") + "\n\n"
	}

	var mentions []string
	var mentionText strings.Builder

	mentionText.WriteString(extra)
	mentionText.WriteString("📢 *Tag All*\n\n")

	for i, p := range info.Participants {
		jid := p.JID.String()
		mentions = append(mentions, jid)
		mentionText.WriteString(fmt.Sprintf("%d. @%s\n", i+1, p.JID.User))
	}

	text := mentionText.String()
	_, err = wac.SendMessage(
		context.Background(),
		v.Info.Chat,
		&waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: &text,
				ContextInfo: &waProto.ContextInfo{
					MentionedJID: mentions,
				},
			},
		},
	)
	return err
}

// ── KICK ──────────────────────────────────────────────────────────────────────
func kickHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	target := resolveTarget(v, args)
	if target == "" {
		return ReplyErr(v, "❌ Reply to user or: .kick 9198xxxxxx")
	}
	targetJID, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID: "+target)
	}
	wac := client.Get()
	_, err = wac.UpdateGroupParticipants(context.Background(), v.Info.Chat, []types.JID{targetJID}, "remove")
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Kick failed: %v", err))
	}
	user := strings.Split(target, "@")[0]
	return ReplyErr(v, fmt.Sprintf("✅ Kicked: *+%s*", user))
}

// ── ADD ───────────────────────────────────────────────────────────────────────
func addMemberHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	if len(args) == 0 {
		return ReplyErr(v, "❌ Use: .add 9198xxxxxx")
	}
	num := args[0]
	if !strings.Contains(num, "@") {
		num += "@s.whatsapp.net"
	}
	targetJID, err := types.ParseJID(num)
	if err != nil {
		return ReplyErr(v, "❌ Invalid number.")
	}
	wac := client.Get()
	_, err = wac.UpdateGroupParticipants(context.Background(), v.Info.Chat, []types.JID{targetJID}, "add")
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Add failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("✅ Added: *+%s*", args[0]))
}

// ── PROMOTE ───────────────────────────────────────────────────────────────────
func promoteHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	target := resolveTarget(v, args)
	if target == "" {
		return ReplyErr(v, "❌ Reply to user or: .promote 9198xxxxxx")
	}
	targetJID, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}
	wac := client.Get()
	_, err = wac.UpdateGroupParticipants(context.Background(), v.Info.Chat, []types.JID{targetJID}, "promote")
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Promote failed: %v", err))
	}
	user := strings.Split(target, "@")[0]
	return ReplyErr(v, fmt.Sprintf("✅ Promoted to admin: *+%s*", user))
}

// ── DEMOTE ────────────────────────────────────────────────────────────────────
func demoteHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	target := resolveTarget(v, args)
	if target == "" {
		return ReplyErr(v, "❌ Reply to user or: .demote 9198xxxxxx")
	}
	targetJID, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}
	wac := client.Get()
	_, err = wac.UpdateGroupParticipants(context.Background(), v.Info.Chat, []types.JID{targetJID}, "demote")
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Demote failed: %v", err))
	}
	user := strings.Split(target, "@")[0]
	return ReplyErr(v, fmt.Sprintf("✅ Demoted: *+%s*", user))
}

// ── GROUP INFO ────────────────────────────────────────────────────────────────
func groupInfoHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
	info, err := wac.GetGroupInfo(context.Background(), v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	admins := 0
	for _, p := range info.Participants {
		if p.IsAdmin || p.IsSuperAdmin {
			admins++
		}
	}
	msg := fmt.Sprintf(
		"╔══════════════════════╗\n"+
			"║   📋 *GROUP INFO*     ║\n"+
			"╚══════════════════════╝\n\n"+
			"📌 *Name:* %s\n"+
			"🆔 *JID:* %s\n"+
			"👥 *Members:* %d\n"+
			"👑 *Admins:* %d\n"+
			"📅 *Created:* %s\n"+
			"📝 *Desc:* %s",
		info.Name,
		info.JID.String(),
		len(info.Participants),
		admins,
		info.GroupCreated.Format("02 Jan 2006"),
		info.Topic,
	)
	return ReplyErr(v, msg)
}


// REGISTER
func init() {
	// Restore persisted gbanned entries from MongoDB.
	loaded := database.LoadAllGBanned()
	gbannedMu.Lock()
	for groupJID, users := range loaded {
		if gbanned[groupJID] == nil {
			gbanned[groupJID] = make(map[string]bool)
		}
		for userJID := range users {
			gbanned[groupJID][userJID] = true
		}
	}
	gbannedMu.Unlock()

	Register(ModuleInfo{
		Name:        "GroupTools",
		Description: "Group management — gmute, gban, tagall, kick, promote, demote",
		Commands: []CommandInfo{
			{Pattern: "kick",      Handler: kickHandler,      Sudo: true},
			{Pattern: "add",       Handler: addMemberHandler, Sudo: true},
			{Pattern: "promote",   Handler: promoteHandler,   Sudo: true},
			{Pattern: "demote",    Handler: demoteHandler,    Sudo: true},
			{Pattern: "groupinfo", Handler: groupInfoHandler, Sudo: true},
			{Pattern: "gmute",     Handler: gmuteHandler,     Sudo: true},
			{Pattern: "gunmute",   Handler: gunmuteHandler,   Sudo: true},
			{Pattern: "gmutechat",  Handler: gmuteHandler,   Sudo: true},
			{Pattern: "gunmutechat",Handler: gunmuteHandler, Sudo: true},
			{Pattern: "gtag",       Handler: tagAllGroupHandler,Sudo: true},
			{Pattern: "gban",      Handler: gbanHandler,      Sudo: true},
			{Pattern: "ungban",    Handler: ungbanHandler,    Sudo: true},
			{Pattern: "gbanlist",  Handler: gbanListHandler,  Sudo: true},
		},
	})
}
