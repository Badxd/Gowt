package modules

import (
	"fmt"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// Ban system: kick and prevent re-add via ban list.
//
// .ban @user      — kick + add to ban list
// .unban @user    — remove from ban list
// .banlist        — show banned users in this chat

var (
	banMu   sync.RWMutex
	banList = make(map[string][]string) // chatJID -> []normalizedUserJID
)

// IsBanned checks if a user JID is banned in a chat.
func IsBanned(chat, userJID string) bool {
	norm := normalizeJIDForSudo(userJID)
	banMu.RLock()
	list := banList[chat]
	banMu.RUnlock()
	for _, b := range list {
		if b == norm {
			return true
		}
	}
	return false
}

// extractTargetJID extracts the target user JID from args (mention or phone number).
// Returns the JID string (e.g. "628xxx@s.whatsapp.net") or error.
func extractTargetJID(v *events.Message, args []string) (string, error) {
	// Check quoted message first
	ctxInfo := v.Message.GetExtendedTextMessage().GetContextInfo()
	if ctxInfo != nil && ctxInfo.GetParticipant() != "" {
		return ctxInfo.GetParticipant(), nil
	}

	// Check @mentions in context info
	if ctxInfo != nil && len(ctxInfo.GetMentionedJID()) > 0 {
		return ctxInfo.GetMentionedJID()[0], nil
	}

	// Check args for phone number or @mention
	if len(args) > 0 {
		arg := args[0]
		arg = strings.TrimPrefix(arg, "@")
		arg = strings.TrimPrefix(arg, "+")
		// strip non-digits
		digits := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, arg)
		if len(digits) >= 7 {
			return digits + "@s.whatsapp.net", nil
		}
	}

	return "", fmt.Errorf("no target found")
}

func banHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .ban @mention or reply to a message")
	}

	chat := v.Info.Chat.String()
	norm := normalizeJIDForSudo(target)

	// Add to ban list
	banMu.Lock()
	already := false
	for _, b := range banList[chat] {
		if b == norm {
			already = true
			break
		}
	}
	if !already {
		banList[chat] = append(banList[chat], norm)
	}
	banMu.Unlock()

	// Kick from group
	wac := getClientSafe()
	kickMsg := ""
	if wac != nil {
		targetJID, parseErr := types.ParseJID(target)
		if parseErr == nil {
			ctx, cancel := newContext(15)
			resp, kickErr := wac.UpdateGroupParticipants(ctx, v.Info.Chat, []types.JID{targetJID}, "remove")
			cancel()
			_ = resp
			if kickErr != nil {
				kickMsg = fmt.Sprintf("\n⚠️ Could not kick: %v", kickErr)
			}
		}
	}

	phone := strings.Split(norm, "@")[0]
	return ReplyErr(v, fmt.Sprintf("🚫 *+%s* has been *banned* and kicked from this group.%s", phone, kickMsg))
}

func unbanHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .unban @mention or reply to a message")
	}

	chat := v.Info.Chat.String()
	norm := normalizeJIDForSudo(target)

	banMu.Lock()
	list := banList[chat]
	newList := list[:0]
	found := false
	for _, b := range list {
		if b == norm {
			found = true
			continue
		}
		newList = append(newList, b)
	}
	banList[chat] = newList
	banMu.Unlock()

	phone := strings.Split(norm, "@")[0]
	if !found {
		return ReplyErr(v, fmt.Sprintf("ℹ️ *+%s* is not in the ban list.", phone))
	}
	return ReplyErr(v, fmt.Sprintf("✅ *+%s* has been *unbanned*.", phone))
}

func banListHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()

	banMu.RLock()
	list := banList[chat]
	banMu.RUnlock()

	if len(list) == 0 {
		return ReplyErr(v, "✅ No banned users in this chat.")
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("🚫 *Banned Users (%d):*", len(list)), "")
	for i, norm := range list {
		phone := strings.Split(norm, "@")[0]
		lines = append(lines, fmt.Sprintf("%d. +%s", i+1, phone))
	}
	return ReplyErr(v, strings.Join(lines, "\n"))
}

func init() {
	Register(ModuleInfo{
		Name:        "Ban",
		Description: "Kick and ban users from groups, with ban list enforcement",
		Commands: []CommandInfo{
			{Pattern: "ban", Handler: banHandler, Sudo: true},
			{Pattern: "unban", Handler: unbanHandler, Sudo: true},
			{Pattern: "banlist", Handler: banListHandler, Sudo: true},
		},
	})
}
