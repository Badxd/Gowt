package modules

import (
	"fmt"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow/types/events"
)

// ─── Echo store: user JID → chat JID (echo this user in this chat) ────────
var (
	echoMu    sync.RWMutex
	echoStore = make(map[string]string) // senderJID → chatJID
)

func echoAdd(senderJID, chatJID string) {
	echoMu.Lock()
	defer echoMu.Unlock()
	echoStore[senderJID] = chatJID
}

func echoRemove(senderJID string) {
	echoMu.Lock()
	defer echoMu.Unlock()
	delete(echoStore, senderJID)
}

func echoCheck(senderJID, chatJID string) bool {
	echoMu.RLock()
	defer echoMu.RUnlock()
	c, ok := echoStore[senderJID]
	return ok && c == chatJID
}

func echoAll() map[string]string {
	echoMu.RLock()
	defer echoMu.RUnlock()
	out := make(map[string]string, len(echoStore))
	for k, v := range echoStore {
		out[k] = v
	}
	return out
}

// ─── Helpers ──────────────────────────────────────────────────────────────

func getQuotedSenderJID(v *events.Message) string {
	if v.Message == nil || v.Message.ExtendedTextMessage == nil {
		return ""
	}
	ctx := v.Message.ExtendedTextMessage.ContextInfo
	if ctx == nil {
		return ""
	}
	p := ctx.GetParticipant()
	if p == "" {
		return ""
	}
	return p
}

// ─── Handlers ─────────────────────────────────────────────────────────────

func echoOnHandler(v *events.Message, args []string) error {
	senderJID := getQuotedSenderJID(v)
	if senderJID == "" {
		return ReplyErr(v, "Reply to a user's message to enable echo for them.")
	}
	chatJID := v.Info.Chat.String()
	if echoCheck(senderJID, chatJID) {
		return ReplyErr(v, "Echo is already enabled for this user.")
	}
	echoAdd(senderJID, chatJID)
	return ReplyErr(v, fmt.Sprintf("✅ Echo enabled for `%s`", senderJID))
}

func echoOffHandler(v *events.Message, args []string) error {
	senderJID := getQuotedSenderJID(v)
	if senderJID == "" {
		return ReplyErr(v, "Reply to a user's message to disable echo.")
	}
	chatJID := v.Info.Chat.String()
	if !echoCheck(senderJID, chatJID) {
		return ReplyErr(v, "Echo was not enabled for this user.")
	}
	echoRemove(senderJID)
	return ReplyErr(v, "✅ Echo disabled for this user.")
}

func echoListHandler(v *events.Message, args []string) error {
	all := echoAll()
	if len(all) == 0 {
		return ReplyErr(v, "No users currently echoed.")
	}
	var sb strings.Builder
	sb.WriteString("🔁 *Echoed Users:*\n\n")
	for sender, chat := range all {
		sb.WriteString(fmt.Sprintf("• `%s` in `%s`\n", sender, chat))
	}
	return ReplyErr(v, sb.String())
}

func init() {
	Register(ModuleInfo{
		Name:        "Echo",
		Description: "Echo (repeat) every message from a specific user",
		Commands: []CommandInfo{
			{Pattern: "echo", Handler: echoOnHandler, Sudo: true},
			{Pattern: "rmecho", Handler: echoOffHandler, Sudo: true},
			{Pattern: "listecho", Handler: echoListHandler, Sudo: true},
		},
	})
}
