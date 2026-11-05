package modules

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// Warning system: give users warnings, auto-kick after threshold.
//
// .warn @user [reason]   — give a warning (default threshold: 3 warns = kick)
// .warns @user           — show warning count
// .resetwarn @user       — reset warnings
// .setwarnlimit <n>      — set warn limit for this chat (default 3)
// .warnlist              — show all warned users in this chat

type warnEntry struct {
	Count   int
	Reasons []string
	LastAt  time.Time
}

var (
	warnMu    sync.RWMutex
	warnMap   = make(map[string]*warnEntry) // "chatJID|userJID" -> entry
	warnLimit = make(map[string]int)        // chatJID -> limit (default 3)
)

func getWarnLimit(chat string) int {
	warnMu.RLock()
	limit, ok := warnLimit[chat]
	warnMu.RUnlock()
	if !ok || limit <= 0 {
		return 3
	}
	return limit
}

func warnHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .warn @mention [reason]")
	}

	// reason = everything after the mention/number arg
	reason := "No reason provided"
	if len(args) > 1 {
		reason = strings.Join(args[1:], " ")
	}

	chat := v.Info.Chat.String()
	normalized := normalizeJIDForSudo(target)
	key := chat + "|" + normalized
	limit := getWarnLimit(chat)

	warnMu.Lock()
	entry, ok := warnMap[key]
	if !ok {
		entry = &warnEntry{}
		warnMap[key] = entry
	}
	entry.Count++
	entry.Reasons = append(entry.Reasons, fmt.Sprintf("%d. %s", entry.Count, reason))
	entry.LastAt = time.Now()
	count := entry.Count
	warnMu.Unlock()

	phone := strings.Split(normalized, "@")[0]

	if count >= limit {
		// auto-kick
		wac := getClientSafe()
		if wac != nil {
			targetJID, parseErr := types.ParseJID(target)
			if parseErr == nil {
				ctx, cancel := newContext(15)
				_, _ = wac.UpdateGroupParticipants(ctx, v.Info.Chat, []types.JID{targetJID}, "remove")
				cancel()
			}
		}
		// reset warns after kick
		warnMu.Lock()
		delete(warnMap, key)
		warnMu.Unlock()
		return ReplyErr(v, fmt.Sprintf(
			"⚠️ *+%s* has received warning *%d/%d*.\n📛 Reason: %s\n\n🚫 *Kicked!* Maximum warnings reached.",
			phone, count, limit, reason))
	}

	return ReplyErr(v, fmt.Sprintf(
		"⚠️ *+%s* has been warned.\n📛 Reason: %s\n📊 Warnings: *%d/%d*",
		phone, reason, count, limit))
}

func warnsHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .warns @mention or .warns <number>")
	}

	chat := v.Info.Chat.String()
	normalized := normalizeJIDForSudo(target)
	key := chat + "|" + normalized
	limit := getWarnLimit(chat)

	warnMu.RLock()
	entry := warnMap[key]
	warnMu.RUnlock()

	phone := strings.Split(normalized, "@")[0]
	if entry == nil || entry.Count == 0 {
		return ReplyErr(v, fmt.Sprintf("✅ *+%s* has no warnings.", phone))
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("⚠️ *Warnings for +%s:* %d/%d", phone, entry.Count, limit), "")
	for _, r := range entry.Reasons {
		lines = append(lines, "  • "+r)
	}
	return ReplyErr(v, strings.Join(lines, "\n"))
}

func resetWarnHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .resetwarn @mention or .resetwarn <number>")
	}

	chat := v.Info.Chat.String()
	normalized := normalizeJIDForSudo(target)
	key := chat + "|" + normalized

	warnMu.Lock()
	_, existed := warnMap[key]
	delete(warnMap, key)
	warnMu.Unlock()

	phone := strings.Split(normalized, "@")[0]
	if !existed {
		return ReplyErr(v, fmt.Sprintf("ℹ️ *+%s* had no warnings.", phone))
	}
	return ReplyErr(v, fmt.Sprintf("✅ Warnings reset for *+%s*.", phone))
}

func setWarnLimitHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .setwarnlimit <number>\nExample: .setwarnlimit 5")
	}
	var n int
	if _, err := fmt.Sscan(args[0], &n); err != nil || n < 1 || n > 20 {
		return ReplyErr(v, "❌ Invalid number. Must be between 1 and 20.")
	}
	chat := v.Info.Chat.String()
	warnMu.Lock()
	warnLimit[chat] = n
	warnMu.Unlock()
	return ReplyErr(v, fmt.Sprintf("✅ Warn limit set to *%d*. Users will be kicked after %d warnings.", n, n))
}

func warnListHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()
	prefix := chat + "|"
	limit := getWarnLimit(chat)

	warnMu.RLock()
	defer warnMu.RUnlock()

	var lines []string
	lines = append(lines, fmt.Sprintf("⚠️ *Warned Users (limit: %d):*", limit), "")

	count := 0
	for key, entry := range warnMap {
		if !strings.HasPrefix(key, prefix) || entry.Count == 0 {
			continue
		}
		user := strings.TrimPrefix(key, prefix)
		phone := strings.Split(user, "@")[0]
		lines = append(lines, fmt.Sprintf("  • +%s — %d/%d warns", phone, entry.Count, limit))
		count++
	}

	if count == 0 {
		return ReplyErr(v, "✅ No users have active warnings in this chat.")
	}
	return ReplyErr(v, strings.Join(lines, "\n"))
}

func init() {
	Register(ModuleInfo{
		Name:        "Warnings",
		Description: "Warning system with auto-kick after threshold",
		Commands: []CommandInfo{
			{Pattern: "warn", Handler: warnHandler, Sudo: true},
			{Pattern: "warns", Handler: warnsHandler, Sudo: true},
			{Pattern: "resetwarn", Handler: resetWarnHandler, Sudo: true},
			{Pattern: "setwarnlimit", Handler: setWarnLimitHandler, Sudo: true},
			{Pattern: "warnlimit", Handler: setWarnLimitHandler, Sudo: true},
			{Pattern: "warnlist", Handler: warnListHandler, Sudo: true},
		},
	})
}
