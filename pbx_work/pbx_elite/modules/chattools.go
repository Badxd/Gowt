package modules

// Chat Tools — archive, pin, disappearing messages, mute/unmute chats.
//
// Commands:
//   .archive              — archive current chat
//   .unarchive            — unarchive current chat
//   .pin                  — pin current chat
//   .unpin                — unpin current chat
//   .mute [hours]         — mute this chat (default 8 hours)
//   .unmute               — unmute this chat
//   .disappear [off|24h|7d|90d] — set disappearing messages timer
//   .chatinfo             — show current chat metadata

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"

	"PbxGb/client"
)

// ── ARCHIVE / UNARCHIVE ───────────────────────────────────────────────────────
func archiveChatHandler(v *events.Message, args []string) error {
	return setArchive(v, true)
}

func unarchiveChatHandler(v *events.Message, args []string) error {
	return setArchive(v, false)
}

func setArchive(v *events.Message, archive bool) error {
	wac := client.Get()
	ctx := context.Background()

	ts := uint64(v.Info.Timestamp.Unix())
	msgID := v.Info.ID
	chat := v.Info.Chat.String()

	patch, err := wac.Store.AppState.NewPatch(appstate.WAPatchRegular, []appstate.PatchInfo{
		{
			Type:  appstate.WAPatchRegular,
			Index: []string{"archive", chat},
			ActionValue: &waProto.SyncActionValue{
				ArchiveChatAction: &waProto.ArchiveChatAction{
					Archived: proto.Bool(archive),
					MessageRange: &waProto.SyncActionMessageRange{
						LastMessageTimestamp: proto.Int64(int64(ts)),
						Messages: []*waProto.SyncActionMessage{
							{Key: &waProto.MessageKey{
								RemoteJid: proto.String(chat),
								Id:        proto.String(msgID),
							}},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Patch error: %v", err))
	}
	if err := wac.SendAppState(ctx, patch); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ AppState sync failed: %v", err))
	}
	if archive {
		return ReplyErr(v, "📦 Chat archived.")
	}
	return ReplyErr(v, "📂 Chat unarchived.")
}

// ── PIN / UNPIN ───────────────────────────────────────────────────────────────
func pinChatHandler(v *events.Message, args []string) error {
	return setPinChat(v, true)
}

func unpinChatHandler(v *events.Message, args []string) error {
	return setPinChat(v, false)
}

func setPinChat(v *events.Message, pin bool) error {
	wac := client.Get()
	ctx := context.Background()
	chat := v.Info.Chat.String()
	now := uint64(time.Now().Unix())

	patch, err := wac.Store.AppState.NewPatch(appstate.WAPatchRegular, []appstate.PatchInfo{
		{
			Type:  appstate.WAPatchRegular,
			Index: []string{"pin_v1", chat},
			ActionValue: &waProto.SyncActionValue{
				PinAction: &waProto.PinAction{
					Pinned: proto.Bool(pin),
				},
				Timestamp: proto.Int64(int64(now)),
			},
		},
	})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Patch error: %v", err))
	}
	if err := wac.SendAppState(ctx, patch); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	if pin {
		return ReplyErr(v, "📌 Chat pinned.")
	}
	return ReplyErr(v, "📍 Chat unpinned.")
}

// ── MUTE / UNMUTE ─────────────────────────────────────────────────────────────
func muteChatHandler(v *events.Message, args []string) error {
	hours := 8
	if len(args) > 0 {
		if h, err := strconv.Atoi(args[0]); err == nil && h > 0 {
			hours = h
		}
	}
	until := time.Now().Add(time.Duration(hours) * time.Hour)
	return setMuteChat(v, true, until)
}

func unmuteChatHandler(v *events.Message, args []string) error {
	return setMuteChat(v, false, time.Time{})
}

func setMuteChat(v *events.Message, mute bool, until time.Time) error {
	wac := client.Get()
	ctx := context.Background()
	chat := v.Info.Chat.String()

	var muteEndTime int64
	if mute && !until.IsZero() {
		muteEndTime = until.Unix()
	}

	patch, err := wac.Store.AppState.NewPatch(appstate.WAPatchRegular, []appstate.PatchInfo{
		{
			Type:  appstate.WAPatchRegular,
			Index: []string{"mute", chat},
			ActionValue: &waProto.SyncActionValue{
				MuteAction: &waProto.MuteAction{
					Muted:          proto.Bool(mute),
					MuteEndTimestamp: proto.Int64(muteEndTime),
				},
			},
		},
	})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Patch error: %v", err))
	}
	if err := wac.SendAppState(ctx, patch); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	if mute {
		return ReplyErr(v, fmt.Sprintf("🔇 Chat muted until %s.", until.Format("02 Jan 3:04 PM")))
	}
	return ReplyErr(v, "🔔 Chat unmuted.")
}

// ── DISAPPEARING MESSAGES ─────────────────────────────────────────────────────
func disappearHandler(v *events.Message, args []string) error {
	wac := client.Get()
	ctx := context.Background()

	// Parse timer
	timerArg := "24h"
	if len(args) > 0 {
		timerArg = strings.ToLower(args[0])
	}

	var timer types.DisappearingTimerDuration
	var label string
	switch timerArg {
	case "off", "0", "disable":
		timer = 0
		label = "OFF"
	case "24h", "1d", "day":
		timer = types.DisappearingTimer24Hours
		label = "24 hours"
	case "7d", "7", "week":
		timer = types.DisappearingTimer7Days
		label = "7 days"
	case "90d", "90", "3m":
		timer = types.DisappearingTimer90Days
		label = "90 days"
	default:
		return ReplyErr(v, "Usage: .disappear [off|24h|7d|90d]")
	}

	// For DM chats
	if !isGroup(v) {
		if err := wac.SetDisappearingTimer(ctx, v.Info.Chat, timer); err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
		}
		if timer == 0 {
			return ReplyErr(v, "✅ Disappearing messages disabled.")
		}
		return ReplyErr(v, fmt.Sprintf("⏱️ Disappearing messages set to *%s*.", label))
	}

	// For groups — set via group update
	if err := wac.SetGroupDisappearingTimer(ctx, v.Info.Chat, timer); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	if timer == 0 {
		return ReplyErr(v, "✅ Disappearing messages disabled in this group.")
	}
	return ReplyErr(v, fmt.Sprintf("⏱️ Disappearing messages set to *%s* in this group.", label))
}

// ── CHAT INFO ─────────────────────────────────────────────────────────────────
func chatInfoHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat
	chatType := "DM Chat"
	if isGroup(v) {
		chatType = "Group"
	}

	wac := client.Get()
	name := chat.User
	extra := ""

	if isGroup(v) {
		if info, err := wac.GetGroupInfo(chat); err == nil {
			name = info.Name
			extra = fmt.Sprintf(
				"\n👥 *Members:* %d\n"+
					"📅 *Created:* %s\n"+
					"🔒 *Locked:* %v\n"+
					"📢 *Announce:* %v",
				len(info.Participants),
				info.GroupCreated.Format("02 Jan 2006"),
				info.IsLocked,
				info.IsAnnounce,
			)
		}
	}

	return ReplyErr(v, fmt.Sprintf(
		"ℹ️ *Chat Info*\n\n"+
			"🏷️ *Name:* %s\n"+
			"🆔 *JID:* `%s`\n"+
			"📱 *Type:* %s%s",
		name, chat.String(), chatType, extra,
	))
}

func init() {
	Register(ModuleInfo{
		Name:        "ChatTools",
		Description: "Archive, pin, mute, disappearing messages, chat info",
		Commands: []CommandInfo{
			{Pattern: "archive",   Handler: archiveChatHandler,  Sudo: true},
			{Pattern: "unarchive", Handler: unarchiveChatHandler,Sudo: true},
			{Pattern: "pin",       Handler: pinChatHandler,      Sudo: true},
			{Pattern: "unpin",     Handler: unpinChatHandler,    Sudo: true},
			{Pattern: "mute",      Handler: muteChatHandler,     Sudo: true},
			{Pattern: "unmute",    Handler: unmuteChatHandler,   Sudo: true},
			{Pattern: "disappear", Handler: disappearHandler,    Sudo: true},
			{Pattern: "vanish",    Handler: disappearHandler,    Sudo: true},
			{Pattern: "chatinfo",  Handler: chatInfoHandler,     Sudo: true},
		},
	})
}
