package modules

// Broadcast Tools — manage broadcast lists and bulk messaging.
//
// Commands:
//   .broadcast <text>     — send message to all saved contacts (DM only, not groups)
//   .groupbroadcast <msg> — send message to all groups you're in
//   .tagall               — mention all members in current group
//   .tagadmins            — mention only admins in current group
//   .announce <text>      — send as announcement (admin-only groups: unlock, send, lock)

import (
	"context"
	"fmt"
	"strings"
	"time"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"PbxGb/client"
)

// ── BROADCAST TO CONTACTS ─────────────────────────────────────────────────────
func broadcastHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .broadcast <message>")
	}
	text := strings.Join(args, " ")
	wac := client.Get()
	ctx := context.Background()

	// Get all saved contacts from store
	allContacts, err := wac.Store.Contacts.GetAllContacts()
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to get contacts: %v", err))
	}

	sent, skipped := 0, 0
	for jid := range allContacts {
		if jid.Server != types.DefaultUserServer {
			continue // skip groups, newsletter, etc
		}
		if wac.Store.ID != nil && jid.User == wac.Store.ID.User {
			continue // skip self
		}
		_, err := wac.SendMessage(ctx, jid, &waProto.Message{
			Conversation: proto.String(text),
		})
		if err != nil {
			skipped++
		} else {
			sent++
		}
		if sent >= 30 { // safety cap — avoid spam bans
			break
		}
		time.Sleep(500 * time.Millisecond) // small delay between sends
	}
	return ReplyErr(v, fmt.Sprintf(
		"📢 *Broadcast done*\n✅ Sent: %d\n❌ Skipped: %d\n_(capped at 30 contacts)_",
		sent, skipped,
	))
}

// ── BROADCAST TO ALL GROUPS ───────────────────────────────────────────────────
func groupBroadcastHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .groupbroadcast <message>")
	}
	text := strings.Join(args, " ")
	wac := client.Get()
	ctx := context.Background()

	groups, err := wac.GetJoinedGroups()
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to get groups: %v", err))
	}

	sent, skipped := 0, 0
	for _, g := range groups {
		_, err := wac.SendMessage(ctx, g.JID, &waProto.Message{
			Conversation: proto.String(text),
		})
		if err != nil {
			skipped++
		} else {
			sent++
		}
		if sent >= 20 { // cap
			break
		}
		time.Sleep(800 * time.Millisecond)
	}
	return ReplyErr(v, fmt.Sprintf(
		"📢 *Group broadcast done*\n✅ Sent to: %d groups\n❌ Skipped: %d\n_(capped at 20 groups)_",
		sent, skipped,
	))
}

// ── TAG ALL ───────────────────────────────────────────────────────────────────
func tagAllHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
	info, err := wac.GetGroupInfo(v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	msg := "📣 Attention everyone!"
	if len(args) > 0 {
		msg = strings.Join(args, " ")
	}

	var tags []string
	for _, p := range info.Participants {
		tags = append(tags, "@"+p.JID.User)
	}

	full := msg + "\n\n" + strings.Join(tags, " ")
	sendTextWithMentions(wac, v.Info.Chat, full, info.Participants)
	return nil
}

// ── TAG ADMINS ────────────────────────────────────────────────────────────────
func tagAdminsHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
	info, err := wac.GetGroupInfo(v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	msg := "📣 Admins, attention!"
	if len(args) > 0 {
		msg = strings.Join(args, " ")
	}

	var adminParticipants []types.GroupParticipant
	var tags []string
	for _, p := range info.Participants {
		if p.IsAdmin || p.IsSuperAdmin {
			adminParticipants = append(adminParticipants, p)
			tags = append(tags, "@"+p.JID.User)
		}
	}

	if len(adminParticipants) == 0 {
		return ReplyErr(v, "ℹ️ No admins found.")
	}

	full := msg + "\n\n" + strings.Join(tags, " ")
	sendTextWithMentions(wac, v.Info.Chat, full, adminParticipants)
	return nil
}

// ── ANNOUNCE (unlock → send → lock) ──────────────────────────────────────────
func announceHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .announce <message>")
	}
	text := strings.Join(args, " ")
	wac := client.Get()
	ctx := context.Background()

	// 1. Unlock group so bot can send
	_ = wac.SetGroupAnnounce(ctx, v.Info.Chat, false)

	// 2. Send the message
	_, err := wac.SendMessage(ctx, v.Info.Chat, &waProto.Message{
		Conversation: proto.String("📢 *ANNOUNCEMENT*\n\n" + text),
	})

	// 3. Re-lock group
	_ = wac.SetGroupAnnounce(ctx, v.Info.Chat, true)

	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to send announcement: %v", err))
	}
	return nil // message already sent to group
}

func init() {
	Register(ModuleInfo{
		Name:        "BroadcastTools",
		Description: "Broadcast to contacts/groups, tag all/admins, group announcement",
		Commands: []CommandInfo{
			{Pattern: "broadcast",       Handler: broadcastHandler,       Sudo: true},
			{Pattern: "groupbroadcast",  Handler: groupBroadcastHandler,  Sudo: true},
			{Pattern: "gbroadcast",      Handler: groupBroadcastHandler,  Sudo: true},
			{Pattern: "tagall",          Handler: tagAllHandler,          Sudo: true},
			{Pattern: "everyone",        Handler: tagAllHandler,          Sudo: true},
			{Pattern: "tagadmins",       Handler: tagAdminsHandler,       Sudo: true},
			{Pattern: "pinghimins",      Handler: tagAdminsHandler,       Sudo: true},
			{Pattern: "announce",        Handler: announceHandler,        Sudo: true},
		},
	})
}
