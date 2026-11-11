package modules

import (
	"strings"
	"sync"

	"go.mau.fi/whatsmeow/types/events"
)

// ─── In-memory blacklist store (per chat) ─────────────────────────────────
// key: chatJID string → set of blacklisted words
var (
	blacklistMu    sync.RWMutex
	blacklistStore = make(map[string]map[string]struct{})
)

func blAdd(chatJID, word string) {
	blacklistMu.Lock()
	defer blacklistMu.Unlock()
	if _, ok := blacklistStore[chatJID]; !ok {
		blacklistStore[chatJID] = make(map[string]struct{})
	}
	blacklistStore[chatJID][strings.ToLower(word)] = struct{}{}
}

func blRemove(chatJID, word string) {
	blacklistMu.Lock()
	defer blacklistMu.Unlock()
	if m, ok := blacklistStore[chatJID]; ok {
		delete(m, strings.ToLower(word))
	}
}

func blList(chatJID string) []string {
	blacklistMu.RLock()
	defer blacklistMu.RUnlock()
	m, ok := blacklistStore[chatJID]
	if !ok {
		return nil
	}
	words := make([]string, 0, len(m))
	for w := range m {
		words = append(words, w)
	}
	return words
}

func blCheck(chatJID, text string) bool {
	blacklistMu.RLock()
	defer blacklistMu.RUnlock()
	m, ok := blacklistStore[chatJID]
	if !ok {
		return false
	}
	lower := strings.ToLower(text)
	for word := range m {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}

// ─── Handlers ─────────────────────────────────────────────────────────────

func addBlacklistHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: `.addbl <word1> <word2> ...`")
	}
	chatJID := v.Info.Chat.String()
	for _, w := range args {
		blAdd(chatJID, w)
	}
	return ReplyErr(v, "✅ Added `"+strings.Join(args, ", ")+"` to blacklist.")
}

func rmBlacklistHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: `.rmbl <word1> <word2> ...`")
	}
	chatJID := v.Info.Chat.String()
	for _, w := range args {
		blRemove(chatJID, w)
	}
	return ReplyErr(v, "✅ Removed `"+strings.Join(args, ", ")+"` from blacklist.")
}

func listBlacklistHandler(v *events.Message, args []string) error {
	chatJID := v.Info.Chat.String()
	words := blList(chatJID)
	if len(words) == 0 {
		return ReplyErr(v, "📋 No blacklisted words in this chat.")
	}
	out := "🚫 *Blacklist - This Chat:*\n\n"
	for _, w := range words {
		out += "• `" + w + "`\n"
	}
	return ReplyErr(v, out)
}

func init() {
	Register(ModuleInfo{
		Name:        "Blacklist",
		Description: "Auto-delete messages containing blacklisted words",
		Commands: []CommandInfo{
			{Pattern: "addbl", Handler: addBlacklistHandler, Sudo: true},
			{Pattern: "rmbl", Handler: rmBlacklistHandler, Sudo: true},
			{Pattern: "bllist", Handler: listBlacklistHandler, Sudo: true},
		},
	})
}
