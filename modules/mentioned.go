package modules

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// Tracks when the bot is @mentioned across all chats
// .mentions      — show recent mentions
// .clearmention  — clear mention log

type mentionEntry struct {
	ChatJID   string
	SenderJID string
	Text      string
	Time      time.Time
}

var (
	mentionMu   sync.RWMutex
	mentionLog  []mentionEntry // ring buffer, max 50
)

// TrackMentions is the exported hook called from HandleMessage.
func TrackMentions(v *events.Message) {
	// check if bot's JID is in the mentioned list
	wac := getClientSafe()
	if wac == nil || wac.Store.ID == nil {
		return
	}

	ownJID := wac.Store.ID.String()
	ownUser := strings.Split(ownJID, "@")[0]

	// extract mentioned JIDs from context info
	var mentioned []string
	msg := v.Message

	checkCtxInfo := func(mentions []string) {
		for _, m := range mentions {
			if strings.HasPrefix(m, ownUser+"@") || m == ownJID {
				mentioned = append(mentioned, m)
			}
		}
	}

	if ext := msg.GetExtendedTextMessage(); ext != nil {
		if ci := ext.GetContextInfo(); ci != nil {
			checkCtxInfo(ci.GetMentionedJID())
		}
	}

	// also check body text for @number mention pattern
	body := extractBodyText(v)
	if strings.Contains(body, "@"+ownUser) {
		mentioned = append(mentioned, ownJID)
	}

	if len(mentioned) == 0 {
		return
	}

	senderStr := v.Info.Sender.String()
	chat := v.Info.Chat.String()

	entry := mentionEntry{
		ChatJID:   chat,
		SenderJID: senderStr,
		Text:      body,
		Time:      v.Info.Timestamp,
	}

	mentionMu.Lock()
	mentionLog = append(mentionLog, entry)
	if len(mentionLog) > 50 {
		mentionLog = mentionLog[len(mentionLog)-50:]
	}
	mentionMu.Unlock()
}

// extractBodyText pulls text from whichever message type was sent.
func extractBodyText(v *events.Message) string {
	msg := v.Message
	if t := msg.GetConversation(); t != "" {
		return t
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}
	if img := msg.GetImageMessage(); img != nil {
		return img.GetCaption()
	}
	if vid := msg.GetVideoMessage(); vid != nil {
		return vid.GetCaption()
	}
	return ""
}

func mentionsHandler(v *events.Message, args []string) error {
	mentionMu.RLock()
	entries := make([]mentionEntry, len(mentionLog))
	copy(entries, mentionLog)
	mentionMu.RUnlock()

	if len(entries) == 0 {
		return ReplyErr(v, "ℹ️ No mentions recorded yet.")
	}

	// show last 10, newest first
	start := len(entries) - 10
	if start < 0 {
		start = 0
	}
	recent := entries[start:]

	var lines []string
	lines = append(lines, fmt.Sprintf("📢 *Recent Mentions (%d total)*", len(entries)), "")

	for i := len(recent) - 1; i >= 0; i-- {
		e := recent[i]
		senderPhone := strings.Split(e.SenderJID, "@")[0]
		chatID := strings.Split(e.ChatJID, "@")[0]
		text := e.Text
		if len(text) > 60 {
			text = text[:60] + "…"
		}
		ts := e.Time.Format("02 Jan, 15:04")
		lines = append(lines, fmt.Sprintf("🔔 *%s* in `%s`\n  _%s_\n  🕒 %s", "+"+senderPhone, chatID, text, ts))
		lines = append(lines, "")
	}

	return ReplyErr(v, strings.Join(lines, "\n"))
}

func clearMentionsHandler(v *events.Message, args []string) error {
	mentionMu.Lock()
	mentionLog = nil
	mentionMu.Unlock()

	return ReplyErr(v, "✅ Mention log cleared.")
}

func init() {
	Register(ModuleInfo{
		Name:        "Mentions",
		Description: "Track and view when the bot is @mentioned across all chats",
		Commands: []CommandInfo{
			{Pattern: "mentions", Handler: mentionsHandler, Sudo: true},
			{Pattern: "clearmention", Handler: clearMentionsHandler, Sudo: true},
		},
	})
}
