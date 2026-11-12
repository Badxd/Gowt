package modules

import (
	"fmt"
	"strings"
	"sync"
	"time"


	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"context"
)

// Mute/unmute a specific user in the group (demote from admin + lock them out via group settings)
// Since WhatsApp doesn't support silencing individual users,
// we implement a "soft mute" via bot-tracking + auto-delete their messages
// .mute @user [minutes]  — soft-mute user for n minutes (default permanent)
// .unmute @user          — remove mute
// .mutelist              — list muted users in this chat

type muteEntry struct {
	ChatJID string
	UserJID string
	Until   time.Time // zero = permanent
}

var (
	muteMu   sync.RWMutex
	muteMap  = make(map[string]*muteEntry) // "chatJID|userJID" -> entry
)

// IsMuted checks if a user is currently muted in a chat.
func IsMuted(chatJID, userJID string) bool {
	key := chatJID + "|" + normalizeJIDForSudo(userJID)
	muteMu.RLock()
	entry, ok := muteMap[key]
	muteMu.RUnlock()

	if !ok {
		return false
	}
	// check expiry
	if !entry.Until.IsZero() && time.Now().After(entry.Until) {
		muteMu.Lock()
		delete(muteMap, key)
		muteMu.Unlock()
		return false
	}
	return true
}

// CheckMute is the exported hook — auto-deletes messages from muted users.
func CheckMute(v *events.Message) {
	chat := v.Info.Chat.String()
	sender := v.Info.Sender.String()

	if !IsMuted(chat, sender) {
		return
	}

	wac := getClientSafe()
	if wac == nil {
		return
	}

	// revoke / delete the message
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// it's from them, not us — but RevokeMessage only needs ctx, chat, id
	_, _ = wac.RevokeMessage(ctx, v.Info.Chat, v.Info.ID)
}

func muteHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .mute @mention [minutes]\nExample: .mute @user 30")
	}

	// parse optional duration (last arg if it's a number)
	var until time.Time
	remaining := args
	if len(remaining) > 0 {
		last := remaining[len(remaining)-1]
		var mins int
		if _, err2 := fmt.Sscan(last, &mins); err2 == nil && mins > 0 {
			until = time.Now().Add(time.Duration(mins) * time.Minute)
			remaining = remaining[:len(remaining)-1]
		}
	}

	chat := v.Info.Chat.String()
	normalized := normalizeJIDForSudo(target)
	key := chat + "|" + normalized

	muteMu.Lock()
	muteMap[key] = &muteEntry{
		ChatJID: chat,
		UserJID: normalized,
		Until:   until,
	}
	muteMu.Unlock()

	phone := strings.Split(normalized, "@")[0]
	if until.IsZero() {
		return ReplyErr(v, fmt.Sprintf("🔇 *+%s* has been muted permanently.", phone))
	}
	return ReplyErr(v, fmt.Sprintf("🔇 *+%s* muted for *%s*.", phone, until.Sub(time.Now()).Round(time.Minute)))
}

func unmuteHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .unmute @mention or .unmute <number>")
	}

	chat := v.Info.Chat.String()
	normalized := normalizeJIDForSudo(target)
	key := chat + "|" + normalized

	muteMu.Lock()
	_, existed := muteMap[key]
	delete(muteMap, key)
	muteMu.Unlock()

	phone := strings.Split(normalized, "@")[0]
	if !existed {
		return ReplyErr(v, fmt.Sprintf("⚠️ *+%s* was not muted.", phone))
	}
	return ReplyErr(v, fmt.Sprintf("🔊 *+%s* has been unmuted.", phone))
}

func muteListHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()
	prefix := chat + "|"

	muteMu.RLock()
	defer muteMu.RUnlock()

	var lines []string
	lines = append(lines, "🔇 *Muted Users:*", "")

	count := 0
	for key, entry := range muteMap {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		phone := strings.Split(entry.UserJID, "@")[0]
		if entry.Until.IsZero() {
			lines = append(lines, fmt.Sprintf("  • +%s (permanent)", phone))
		} else if time.Now().Before(entry.Until) {
			remaining := entry.Until.Sub(time.Now()).Round(time.Minute)
			lines = append(lines, fmt.Sprintf("  • +%s (expires in %s)", phone, remaining))
		}
		count++
	}

	if count == 0 {
		return ReplyErr(v, "ℹ️ No users are currently muted in this chat.")
	}

	_ = getClientSafe() // keep import
	return ReplyErr(v, strings.Join(lines, "\n"))
}

func init() {
	Register(ModuleInfo{
		Name:        "Mute",
		Description: "Soft-mute users in a group (auto-delete their messages)",
		Commands: []CommandInfo{
			{Pattern: "mute", Handler: muteHandler, Sudo: true},
			{Pattern: "unmute", Handler: unmuteHandler, Sudo: true},
			{Pattern: "mutelist", Handler: muteListHandler, Sudo: true},
		},
	})
}
