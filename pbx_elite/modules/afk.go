package modules

import (
	"fmt"
	"strings"
	"sync"
	"time"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// AFK (Away From Keyboard): auto-reply when mentioned while AFK.
//
// .afk [reason]   — set yourself as AFK
// .unafk          — clear AFK status (also clears on sending any message)

type afkEntry struct {
	Reason string
	Since  time.Time
}

var (
	afkMu  sync.RWMutex
	afkMap = make(map[string]*afkEntry) // userJID (normalized) -> entry
)

// CheckAFK is the exported hook — auto-replies when an AFK user is mentioned.
func CheckAFK(v *events.Message) {
	wac := getClientSafe()
	if wac == nil || wac.Store.ID == nil {
		return
	}

	senderNorm := normalizeJIDForSudo(v.Info.Sender.String())

	// If the sender is AFK and they just sent a message, clear their AFK status
	afkMu.Lock()
	if _, isAfk := afkMap[senderNorm]; isAfk {
		delete(afkMap, senderNorm)
		afkMu.Unlock()
		ctx, cancel := newContext(8)
		defer cancel()
		msg := &waProto.Message{Conversation: proto.String("✅ Welcome back! Your AFK status has been cleared.")}
		_, _ = wac.SendMessage(ctx, v.Info.Chat, msg)
		return
	}
	afkMu.Unlock()

	// Check if any mentioned user is AFK
	ctxInfo := v.Message.GetExtendedTextMessage().GetContextInfo()
	if ctxInfo == nil {
		return
	}

	for _, mentioned := range ctxInfo.GetMentionedJID() {
		norm := normalizeJIDForSudo(mentioned)
		afkMu.RLock()
		entry, isAfk := afkMap[norm]
		afkMu.RUnlock()

		if !isAfk {
			continue
		}

		elapsed := time.Since(entry.Since).Round(time.Minute)
		reason := entry.Reason
		if reason == "" {
			reason = "No reason given"
		}

		phone := strings.Split(norm, "@")[0]
		text := fmt.Sprintf(
			"😴 *+%s* is currently AFK.\n📝 Reason: %s\n⏱️ Since: %s ago",
			phone, reason, elapsed,
		)

		ctx, cancel := newContext(8)
		replyMsg := &waProto.Message{Conversation: proto.String(text)}
		_, _ = wac.SendMessage(ctx, v.Info.Chat, replyMsg)
		cancel()
	}
}

func afkHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil || wac.Store.ID == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	reason := ""
	if len(args) > 0 {
		reason = strings.Join(args, " ")
	}

	norm := normalizeJIDForSudo(v.Info.Sender.String())

	afkMu.Lock()
	afkMap[norm] = &afkEntry{
		Reason: reason,
		Since:  time.Now(),
	}
	afkMu.Unlock()

	if reason != "" {
		return ReplyErr(v, fmt.Sprintf("😴 You are now AFK.\n📝 Reason: %s\n\nSend any message to clear AFK status.", reason))
	}
	return ReplyErr(v, "😴 You are now AFK. Send any message to clear AFK status.")
}

func unafkHandler(v *events.Message, args []string) error {
	norm := normalizeJIDForSudo(v.Info.Sender.String())

	afkMu.Lock()
	_, was := afkMap[norm]
	delete(afkMap, norm)
	afkMu.Unlock()

	if !was {
		return ReplyErr(v, "ℹ️ You were not AFK.")
	}
	return ReplyErr(v, "✅ AFK status cleared.")
}

func init() {
	Register(ModuleInfo{
		Name:        "AFK",
		Description: "AFK status: auto-reply when mentioned while away",
		Commands: []CommandInfo{
			{Pattern: "afk", Handler: afkHandler, Sudo: false},
			{Pattern: "unafk", Handler: unafkHandler, Sudo: false},
		},
	})
}
