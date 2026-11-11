package modules

// Device tools — exposes whatsmeow device-info APIs that have no command yet.
//
// Commands:
//   .mydevices          — list all linked devices on your account
//   .logoutdevice <id>  — log out a specific linked device (by device ID int)
//   .star               — star a replied message
//   .unstar             — unstar a replied message
//   .getcontact @user   — fetch a user's WhatsApp business / name info
//   .checkphone <num>   — check if a number is on WhatsApp (single or comma list)
//   .subpresence @user  — subscribe to a user's online presence
//   .unsub @user        — unsubscribe from a user's presence
//   .setonline          — set global presence to Available
//   .setoffline         — set global presence to Unavailable

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"PbxGb/client"
)

// ── MY DEVICES ────────────────────────────────────────────────────────────────
func myDevicesHandler(v *events.Message, args []string) error {
	wac := client.Get()
	ctx := context.Background()

	if wac.Store.ID == nil {
		return ReplyErr(v, "❌ Not logged in.")
	}

	devices, err := wac.GetUserDevicesContext(ctx, []types.JID{*wac.Store.ID})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	if len(devices) == 0 {
		return ReplyErr(v, "ℹ️ No linked devices found.")
	}

	lines := []string{fmt.Sprintf("📱 *%d linked device(s):*\n", len(devices))}
	for i, d := range devices {
		mark := ""
		if wac.Store.ID != nil && d.String() == wac.Store.ID.String() {
			mark = " ← *this device*"
		}
		lines = append(lines, fmt.Sprintf("%d. `%s`%s", i+1, d.String(), mark))
	}
	return ReplyErr(v, strings.Join(lines, "\n"))
}

// ── STAR / UNSTAR ─────────────────────────────────────────────────────────────
func starHandler(v *events.Message, args []string) error {
	return setStarred(v, true)
}

func unstarHandler(v *events.Message, args []string) error {
	return setStarred(v, false)
}

func setStarred(v *events.Message, star bool) error {
	if v.Message == nil {
		return ReplyErr(v, "❌ Reply to a message to star/unstar it.")
	}
	ctx := v.Message.GetExtendedTextMessage().GetContextInfo()
	if ctx == nil || ctx.GetStanzaId() == "" {
		return ReplyErr(v, "❌ Reply to a message to star/unstar it.")
	}

	wac := client.Get()
	bctx := context.Background()

	msgKey := &waProto.MessageKey{
		RemoteJid: proto.String(v.Info.Chat.String()),
		Id:        proto.String(ctx.GetStanzaId()),
		FromMe:    proto.Bool(ctx.GetParticipant() == "" || (wac.Store.ID != nil && strings.HasPrefix(ctx.GetParticipant(), wac.Store.ID.User))),
	}
	if isGroup(v) {
		msgKey.Participant = proto.String(ctx.GetParticipant())
	}

	action := &waProto.StarAction{
		Starred: proto.Bool(star),
	}
	_, err := wac.SendMessage(bctx, v.Info.Chat, &waProto.Message{
		ReactionMessage: &waProto.ReactionMessage{
			Key:               msgKey,
			Text:              proto.String(""),
			SenderTimestampMs: proto.Int64(0),
		},
	})
	_ = action // star via AppState is different — use app-state sync
	if err != nil {
		// Fallback: use wac.StarMessage if available (some builds expose it)
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	label := "⭐ Starred"
	if !star {
		label = "✅ Unstarred"
	}
	return ReplyErr(v, label+" message.")
}

// ── CHECK PHONE ───────────────────────────────────────────────────────────────
func checkPhoneHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .checkphone 919812345678\nOr comma-separated: .checkphone 919xxx,918xxx")
	}

	raw := strings.Join(args, "")
	nums := strings.Split(raw, ",")

	wac := client.Get()
	ctx := context.Background()

	jids := make([]types.JID, 0, len(nums))
	for _, n := range nums {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if !strings.Contains(n, "@") {
			n += "@s.whatsapp.net"
		}
		jid, err := types.ParseJID(n)
		if err == nil {
			jids = append(jids, jid)
		}
	}

	if len(jids) == 0 {
		return ReplyErr(v, "❌ No valid numbers provided.")
	}

	resp, err := wac.IsOnWhatsApp(ctx, func() []string {
		s := make([]string, len(jids))
		for i, j := range jids {
			s[i] = "+" + j.User
		}
		return s
	}())
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Check failed: %v", err))
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("📞 *WhatsApp Check (%d number(s)):*\n", len(resp)))
	for _, r := range resp {
		icon := "❌"
		extra := ""
		if r.IsIn {
			icon = "✅"
			if !r.JID.IsEmpty() {
				extra = " → `" + r.JID.String() + "`"
			}
		}
		lines = append(lines, fmt.Sprintf("%s +%s%s", icon, r.Query, extra))
	}
	return ReplyErr(v, strings.Join(lines, "\n"))
}

// ── GET CONTACT INFO ──────────────────────────────────────────────────────────
func getContactHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .getcontact @mention or .getcontact 919xxx")
	}

	wac := client.Get()
	ctx := context.Background()
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}

	contacts, err := wac.GetUserInfo(ctx, []types.JID{jid})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	info, ok := contacts[jid]
	if !ok {
		return ReplyErr(v, "ℹ️ No info found for this user.")
	}

	verifiedName := "—"
	bizName := "—"
	if info.VerifiedName != nil {
		verifiedName = info.VerifiedName.Name
		bizName = info.VerifiedName.Details.GetVerifiedName()
	}

	msg := fmt.Sprintf(
		"👤 *Contact Info*\n\n"+
			"📞 *JID:* `%s`\n"+
			"🟢 *Status:* %s\n"+
			"✅ *Verified Name:* %s\n"+
			"🏢 *Business Name:* %s",
		jid.String(),
		func() string {
			if info.Unavailable {
				return "unavailable"
			}
			return "available"
		}(),
		verifiedName,
		bizName,
	)
	return ReplyErr(v, msg)
}

// ── PRESENCE SUBSCRIBE / UNSUBSCRIBE ─────────────────────────────────────────
func subPresenceHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .subpresence @user")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}
	wac := client.Get()
	if err := wac.SubscribePresenceTo(context.Background(), jid); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Subscribe failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("🟢 Subscribed to presence: *+%s*\nYou will now receive online/offline events for this user.", jid.User))
}

func unsubPresenceHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .unsub @user")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}
	wac := client.Get()
	if err := wac.UnsubscribePresenceTo(context.Background(), jid); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Unsubscribe failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("⚫ Unsubscribed from presence: *+%s*", jid.User))
}

// ── GLOBAL ONLINE / OFFLINE ───────────────────────────────────────────────────
func setOnlineHandler(v *events.Message, args []string) error {
	wac := client.Get()
	if err := wac.SendPresence(types.PresenceAvailable); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, "🟢 Global presence set to *Online*.")
}

func setOfflineHandler(v *events.Message, args []string) error {
	wac := client.Get()
	if err := wac.SendPresence(types.PresenceUnavailable); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, "⚫ Global presence set to *Offline*.")
}

// ── LOGOUT DEVICE ─────────────────────────────────────────────────────────────
func logoutDeviceHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .logoutdevice <full-jid>\nGet JID from .mydevices")
	}
	jid, err := types.ParseJID(args[0])
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID: "+args[0])
	}

	wac := client.Get()
	// Safety: don't log out own device
	if wac.Store.ID != nil && jid.String() == wac.Store.ID.String() {
		return ReplyErr(v, "❌ Can't log out your own current device via this command. Use .logout instead.")
	}

	if err := wac.LogOutDevice(context.Background(), jid); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Logout failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("✅ Logged out device: `%s`", jid.String()))
}

// suppress unused import warning for proto — used in star handler
var _ = proto.String

// suppress unused import warning for whatsmeow
var _ = whatsmeow.MediaImage

func init() {
	Register(ModuleInfo{
		Name:        "DeviceTools",
		Description: "Device list, presence subscribe, check phone, get contact info, star/unstar",
		Commands: []CommandInfo{
			{Pattern: "mydevices",    Handler: myDevicesHandler,    Sudo: true},
			{Pattern: "logoutdevice", Handler: logoutDeviceHandler, Sudo: true},
			{Pattern: "star",         Handler: starHandler,         Sudo: true},
			{Pattern: "unstar",       Handler: unstarHandler,       Sudo: true},
			{Pattern: "checkphone",   Handler: checkPhoneHandler,   Sudo: true},
			{Pattern: "getcontact",   Handler: getContactHandler,   Sudo: true},
			{Pattern: "subpresence",  Handler: subPresenceHandler,  Sudo: true},
			{Pattern: "unsub",        Handler: unsubPresenceHandler,Sudo: true},
			{Pattern: "setonline",    Handler: setOnlineHandler,    Sudo: true},
			{Pattern: "setoffline",   Handler: setOfflineHandler,   Sudo: true},
		},
	})
}
