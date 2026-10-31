package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"PbxGb/client"
	"PbxGb/config"
	"PbxGb/database"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// contextOf returns the ContextInfo of whichever message type carries one.
func contextOf(m *waProto.Message) *waProto.ContextInfo {
	if m == nil {
		return nil
	}
	if x := m.GetExtendedTextMessage(); x != nil {
		return x.GetContextInfo()
	}
	if x := m.GetImageMessage(); x != nil {
		return x.GetContextInfo()
	}
	if x := m.GetVideoMessage(); x != nil {
		return x.GetContextInfo()
	}
	if x := m.GetDocumentMessage(); x != nil {
		return x.GetContextInfo()
	}
	if x := m.GetAudioMessage(); x != nil {
		return x.GetContextInfo()
	}
	if x := m.GetStickerMessage(); x != nil {
		return x.GetContextInfo()
	}
	return nil
}

// mentionsMe — true if the owner is @mentioned or replied to.
func mentionsMe(v *events.Message) bool {
	ci := contextOf(v.Message)
	if ci == nil {
		return false
	}
	wac := client.Get()
	var mine []string
	if wac.Store.ID != nil {
		mine = append(mine, wac.Store.ID.User)
	}
	if wac.Store.LID.User != "" {
		mine = append(mine, wac.Store.LID.User)
	}
	isMine := func(raw string) bool {
		jid, err := types.ParseJID(raw)
		if err != nil {
			return false
		}
		for _, m := range mine {
			if jid.User == m {
				return true
			}
		}
		return false
	}
	for _, j := range ci.GetMentionedJID() {
		if isMine(j) {
			return true
		}
	}
	if p := ci.GetParticipant(); p != "" && isMine(p) {
		return true
	}
	return false
}

// clearAFKOnOwnMessage — when the owner sends ANY message from their phone,
// AFK switches itself off (same as PBX Elite / Telegram userbots).
func clearAFKOnOwnMessage(v *events.Message) {
	if isStale(v) {
		return
	}
	text := strings.ToLower(strings.TrimSpace(extractText(v)))
	p := strings.ToLower(config.C.Prefix)
	if strings.HasPrefix(text, p+"afk") || strings.HasPrefix(text, p+"unafk") {
		return
	}
	state := database.GetAFKState()
	if !state.Enabled {
		return
	}
	if time.Since(state.Since) < 5*time.Second {
		return // AFK was only just switched on
	}
	if err := database.SetAFKState(false, state.Reason, state.Since); err != nil {
		return
	}
	_ = database.ClearAFKNotified()

	msg := fmt.Sprintf("👋 *Welcome back!* AFK is now OFF.\n⏳ You were AFK for *%s*.", formatDuration(time.Since(state.Since)))
	_, _ = client.Get().SendMessage(context.Background(), v.Info.Chat, &waProto.Message{Conversation: &msg})
}

// HandleAFK is called on every message.
//   - owner writes anything  → AFK clears automatically
//   - someone DMs the owner  → one-time notice per sender
//   - owner @mentioned / replied to in a group → one-time notice per sender
func HandleAFK(v *events.Message) {
	if v.Message == nil {
		return
	}
	isDM := v.Info.Chat.Server == types.DefaultUserServer || v.Info.Chat.Server == types.HiddenUserServer
	isGroup := v.Info.Chat.Server == types.GroupServer
	if !isDM && !isGroup {
		return
	}

	if v.Info.IsFromMe {
		clearAFKOnOwnMessage(v)
		return
	}
	if isStale(v) || !isRealMessage(v) {
		return
	}

	wac := client.Get()
	if wac.Store.ID != nil && v.Info.Sender.User == wac.Store.ID.User {
		return
	}
	if wac.Store.LID.User != "" && v.Info.Sender.User == wac.Store.LID.User {
		return
	}

	state := database.GetAFKState()
	if !state.Enabled {
		return
	}
	if isOwner(v) || isSudoSender(v) {
		return
	}
	if isGroup && !mentionsMe(v) {
		return
	}

	key := senderKey(v)
	if isGroup {
		key = v.Info.Chat.String() + "|" + key
	}
	if database.IsAFKNotified(key) {
		return
	}
	_ = database.MarkAFKNotified(key)

	dur := formatDuration(time.Since(state.Since))
	msg := fmt.Sprintf(
		"┌───⭓ 😴 *%s* ⭓───\n"+
			"│ 👋 Hey *%s*!\n"+
			"│ 💤 The owner is *AFK* for *%s*\n"+
			"│ 📝 *Reason:* _%s_\n"+
			"│\n"+
			"│ ⏳ They will reply when they are back.\n"+
			"└───────────────────⭓",
		smallCaps("afk mode"), v.Info.PushName, dur, state.Reason,
	)
	_, _ = Reply(v, msg)
}

// formatDuration converts a duration into a human-readable string.
func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// ── COMMANDS ──────────────────────────────────────────────────────────────────

func afkHandler(v *events.Message, args []string) error {
	reason := "I am AFK right now."
	if len(args) > 0 {
		reason = GetArgs(v)
	}

	if err := database.SetAFKState(true, reason, time.Now()); err != nil {
		return ReplyErr(v, "❌ Failed to save AFK state. Check database connection.")
	}
	_ = database.ClearAFKNotified()

	return ReplyErr(v, fmt.Sprintf(
		"😴 *AFK Mode ON*\n📝 Reason: _%s_\n\nAuto-reply goes to new DMs and @mentions.\nSend any message to switch AFK off.",
		reason,
	))
}

func unAfkHandler(v *events.Message, args []string) error {
	state := database.GetAFKState()
	if !state.Enabled {
		return ReplyErr(v, "ℹ️ AFK mode is not currently active.")
	}
	dur := formatDuration(time.Since(state.Since))

	if err := database.SetAFKState(false, state.Reason, state.Since); err != nil {
		return ReplyErr(v, "❌ Failed to update AFK state. Check database connection.")
	}
	_ = database.ClearAFKNotified()

	return ReplyErr(v, fmt.Sprintf("✅ *AFK Mode OFF*\nYou were AFK for *%s*.", dur))
}

func afkStatusHandler(v *events.Message, args []string) error {
	state := database.GetAFKState()
	if !state.Enabled {
		return ReplyErr(v, "💤 AFK Mode: *OFF*\nUse *"+config.C.Prefix+"afk [reason]* to enable.")
	}
	return ReplyErr(v, fmt.Sprintf(
		"😴 AFK Mode: *ON*\n📝 Reason: _%s_\n⏳ Duration: *%s*",
		state.Reason, formatDuration(time.Since(state.Since)),
	))
}

func init() {
	Register(ModuleInfo{
		Name:        "AFK",
		Description: "Away From Keyboard — DB-backed auto-reply for DMs and mentions, auto-clears when you write",
		Commands: []CommandInfo{
			{Pattern: "afk", Handler: afkHandler, Sudo: true},
			{Pattern: "unafk", Handler: unAfkHandler, Sudo: true},
			{Pattern: "afkstatus", Handler: afkStatusHandler, Sudo: true},
		},
	})
}
