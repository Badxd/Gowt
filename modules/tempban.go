package modules

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"


	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// .tempban @user <minutes>  — kick user and prevent rejoin for N minutes
// .tempbanlist              — show active temp bans in this chat

type tempBanEntry struct {
	ChatJID string
	UserJID string
	Until   time.Time
}

var (
	tempBanMu  sync.RWMutex
	tempBanMap = make(map[string]*tempBanEntry) // "chatJID|userJID" -> entry
)

// IsTempBanned returns true if the user is currently temp-banned from a chat.
// Call this from the group participant join event handler to re-kick.
func IsTempBanned(chatJID, userJID string) bool {
	key := chatJID + "|" + normalizeJIDForSudo(userJID)
	tempBanMu.RLock()
	entry, ok := tempBanMap[key]
	tempBanMu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(entry.Until) {
		tempBanMu.Lock()
		delete(tempBanMap, key)
		tempBanMu.Unlock()
		return false
	}
	return true
}

func tempBanHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .tempban @mention <minutes>\nExample: .tempban @user 30")
	}

	// find duration arg (last numeric arg)
	mins := 30 // default 30 minutes
	for _, a := range args {
		var n int
		if _, e := fmt.Sscan(a, &n); e == nil && n > 0 {
			mins = n
			break
		}
	}
	if mins > 10080 { // max 1 week
		mins = 10080
	}

	until := time.Now().Add(time.Duration(mins) * time.Minute)

	targetJID, err2 := types.ParseJID(target)
	if err2 != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Invalid JID: %v", err2))
	}

	chat := v.Info.Chat.String()
	normalized := normalizeJIDForSudo(target)
	key := chat + "|" + normalized

	tempBanMu.Lock()
	tempBanMap[key] = &tempBanEntry{
		ChatJID: chat,
		UserJID: normalized,
		Until:   until,
	}
	tempBanMu.Unlock()

	// kick the user
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, kickErr := wac.UpdateGroupParticipants(ctx, v.Info.Chat, []types.JID{targetJID}, "remove")

	phone := targetJID.User
	duration := time.Duration(mins) * time.Minute

	if kickErr != nil {
		return ReplyErr(v, fmt.Sprintf("⚠️ Temp ban recorded but kick failed: %v\n⏱️ +%s banned until %s",
			kickErr, phone, until.Format("15:04, 02 Jan")))
	}

	return ReplyErr(v, fmt.Sprintf("⛔ *+%s* has been temp-banned for *%s*.\n⏱️ Ban expires: %s",
		phone, duration.String(), until.Format("15:04, 02 Jan 2006")))
}

func tempBanListHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()
	prefix := chat + "|"

	tempBanMu.RLock()
	defer tempBanMu.RUnlock()

	var lines []string
	lines = append(lines, "⛔ *Active Temp Bans:*", "")

	count := 0
	for key, entry := range tempBanMap {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if time.Now().After(entry.Until) {
			continue
		}
		phone := strings.Split(entry.UserJID, "@")[0]
		remaining := entry.Until.Sub(time.Now()).Round(time.Minute)
		lines = append(lines, fmt.Sprintf("  • +%s → expires in %s", phone, remaining))
		count++
	}

	if count == 0 {
		return ReplyErr(v, "ℹ️ No active temp bans in this chat.")
	}

	return ReplyErr(v, strings.Join(lines, "\n"))
}

func init() {
	Register(ModuleInfo{
		Name:        "Temp Ban",
		Description: "Temporarily ban (kick + track) users from a group",
		Commands: []CommandInfo{
			{Pattern: "tempban", Handler: tempBanHandler, Sudo: true},
			{Pattern: "tban", Handler: tempBanHandler, Sudo: true},
			{Pattern: "tempbanlist", Handler: tempBanListHandler, Sudo: true},
		},
	})
}
