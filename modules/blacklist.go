package modules

import (
	"context"
	"strings"
	"sync"

	"PbxGb/client"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// PBX Elite — Blacklist Module
//
// .addbl <word>   — add word(s) to blacklist in this chat
// .rmbl <word>    — remove word(s)
// .bllist         — show blacklisted words
//
// CheckBlacklist  — hook called from HandleMessage; deletes matching messages

// ─── In-memory blacklist store (per chat) ─────────────────────────────────
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

// getMessageText extracts text content from a message for blacklist checking
func getMessageText(v *events.Message) string {
	if v.Message == nil {
		return ""
	}
	if t := v.Message.GetConversation(); t != "" {
		return t
	}
	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}
	if img := v.Message.GetImageMessage(); img != nil {
		return img.GetCaption()
	}
	if vid := v.Message.GetVideoMessage(); vid != nil {
		return vid.GetCaption()
	}
	if doc := v.Message.GetDocumentMessage(); doc != nil {
		return doc.GetTitle() + " " + doc.GetCaption()
	}
	return ""
}

// CheckBlacklist — hook: delete any message containing a blacklisted word
func CheckBlacklist(v *events.Message) {
	// Don't delete own messages
	if v.Info.IsFromMe {
		return
	}

	chatStr := v.Info.Chat.String()
	text := getMessageText(v)
	if text == "" {
		return
	}
	if !blCheck(chatStr, text) {
		return
	}

	wac := client.Get()
	if wac == nil {
		return
	}

	// Revoke the offending message
	revokeProto := wac.BuildRevoke(
		v.Info.Chat,
		types.EmptyJID,
		v.Info.ID,
	)
	_, _ = wac.SendMessage(context.Background(), v.Info.Chat, revokeProto)
}

// ─── Command Handlers ─────────────────────────────────────────────────────

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
	out := "🚫 *Blacklist — This Chat:*\n\n"
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
