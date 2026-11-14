package modules

import (
	"fmt"
	"strings"
	"sync"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// Word/phrase filter: auto-delete messages containing banned words.
// Per-chat filter lists.
//
// .filter add <word>     — add word to filter
// .filter del <word>     — remove word from filter
// .filter list           — show all filtered words
// .filter clear          — clear all filters for this chat
// .filter on/off         — enable/disable filter for this chat

var (
	filterMu      sync.RWMutex
	filterWords   = make(map[string][]string) // chatJID -> []banned words
	filterEnabled = make(map[string]bool)     // chatJID -> enabled
)

// CheckFilters is the exported hook — auto-deletes messages with banned words.
func CheckFilters(v *events.Message) {
	chat := v.Info.Chat.String()

	filterMu.RLock()
	enabled := filterEnabled[chat]
	words := filterWords[chat]
	filterMu.RUnlock()

	if !enabled || len(words) == 0 {
		return
	}

	wac := getClientSafe()
	if wac == nil {
		return
	}

	// skip own messages
	if wac.Store.ID != nil && strings.HasPrefix(v.Info.Sender.String(), wac.Store.ID.User) {
		return
	}

	body := strings.ToLower(extractBodyText(v))
	if body == "" {
		return
	}

	matched := ""
	for _, w := range words {
		if strings.Contains(body, strings.ToLower(w)) {
			matched = w
			break
		}
	}
	if matched == "" {
		return
	}

	ctx, cancel := newContext(8)
	defer cancel()
	_, _ = wac.RevokeMessage(ctx, v.Info.Chat, v.Info.ID)

	// Warn the user
	mentions := []string{v.Info.Sender.String()}
	warn := fmt.Sprintf("⚠️ @%s, your message was removed for containing a banned word.", v.Info.Sender.User)
	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(warn),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: mentions,
			},
		},
	}
	ctx2, cancel2 := newContext(8)
	defer cancel2()
	_, _ = wac.SendMessage(ctx2, v.Info.Chat, msg)
}

func filterHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	chat := v.Info.Chat.String()

	if len(args) == 0 {
		return ReplyErr(v, "Usage:\n.filter add <word>\n.filter del <word>\n.filter list\n.filter clear\n.filter on/off")
	}

	sub := strings.ToLower(args[0])

	switch sub {
	case "add":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .filter add <word>")
		}
		word := strings.ToLower(strings.Join(args[1:], " "))
		filterMu.Lock()
		// avoid duplicates
		for _, w := range filterWords[chat] {
			if w == word {
				filterMu.Unlock()
				return ReplyErr(v, fmt.Sprintf("ℹ️ *%s* is already in the filter list.", word))
			}
		}
		filterWords[chat] = append(filterWords[chat], word)
		filterEnabled[chat] = true // auto-enable when adding
		filterMu.Unlock()
		return ReplyErr(v, fmt.Sprintf("✅ *%s* added to filter list. Filter is now active.", word))

	case "del", "remove":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .filter del <word>")
		}
		word := strings.ToLower(strings.Join(args[1:], " "))
		filterMu.Lock()
		list := filterWords[chat]
		newList := list[:0]
		found := false
		for _, w := range list {
			if w == word {
				found = true
				continue
			}
			newList = append(newList, w)
		}
		filterWords[chat] = newList
		filterMu.Unlock()
		if !found {
			return ReplyErr(v, fmt.Sprintf("❌ *%s* was not in the filter list.", word))
		}
		return ReplyErr(v, fmt.Sprintf("✅ *%s* removed from filter list.", word))

	case "list":
		filterMu.RLock()
		list := filterWords[chat]
		enabled := filterEnabled[chat]
		filterMu.RUnlock()
		if len(list) == 0 {
			return ReplyErr(v, "ℹ️ No words in the filter list.")
		}
		status := "🔴 disabled"
		if enabled {
			status = "🟢 enabled"
		}
		return ReplyErr(v, fmt.Sprintf("🚫 *Filter (%s) — %d words:*\n%s",
			status, len(list), strings.Join(list, ", ")))

	case "clear":
		filterMu.Lock()
		delete(filterWords, chat)
		filterMu.Unlock()
		return ReplyErr(v, "✅ Filter list cleared.")

	case "on", "enable":
		filterMu.Lock()
		filterEnabled[chat] = true
		filterMu.Unlock()
		return ReplyErr(v, "✅ Word filter *enabled*.")

	case "off", "disable":
		filterMu.Lock()
		filterEnabled[chat] = false
		filterMu.Unlock()
		return ReplyErr(v, "✅ Word filter *disabled*.")

	default:
		return ReplyErr(v, "Usage: .filter add/del/list/clear/on/off")
	}
}

func init() {
	Register(ModuleInfo{
		Name:        "Word Filter",
		Description: "Auto-delete messages containing banned words in a group",
		Commands: []CommandInfo{
			{Pattern: "filter", Handler: filterHandler, Sudo: true},
		},
	})
}
