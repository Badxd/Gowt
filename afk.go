package modules

// PBX Elite — AFK Module
// .afk [reason]  — set AFK
// .back          — clear AFK
// Auto-reply when owner gets a DM or @mention while AFK

import (
	"fmt"
	"strings"
	"sync"
	"time"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

var (
	afkMu        sync.RWMutex
	afkEnabled   = false
	afkReason    = ""
	afkSince     time.Time
	afkLastReply = make(map[string]time.Time) // chatJID → last AFK reply time
)

const afkCooldown = 60 * time.Second // don't double-reply to same chat within 60s

// CheckAFK fires on every incoming message.
// Triggers when:
//   1. Someone @mentions the owner in a group
//   2. Someone sends the owner a direct message (DM)
func CheckAFK(v *events.Message) {
	// Don't react to own messages
	if v.Info.IsFromMe {
		// If owner messages while AFK, clear AFK silently
		afkMu.Lock()
		afkEnabled = false
		afkMu.Unlock()
		return
	}

	afkMu.RLock()
	enabled := afkEnabled
	reason := afkReason
	since := afkSince
	afkMu.RUnlock()

	if !enabled {
		return
	}

	wac := getClientSafe()
	if wac == nil {
		return
	}

	chat := v.Info.Chat.String()
	isGroup := strings.HasSuffix(chat, "@g.us")

	shouldReply := false

	if isGroup {
		// In groups: only reply if owner is @mentioned
		ownerUser := ""
		if wac.Store.ID != nil {
			ownerUser = wac.Store.ID.User
		}
		ctxInfo := v.Message.GetExtendedTextMessage().GetContextInfo()
		if ctxInfo != nil {
			for _, jid := range ctxInfo.GetMentionedJID() {
				if strings.HasPrefix(jid, ownerUser+"@") || jid == ownerUser {
					shouldReply = true
					break
				}
			}
		}
	} else {
		// DM: always reply (someone DMing the owner)
		shouldReply = true
	}

	if !shouldReply {
		return
	}

	// Cooldown: don't spam AFK reply to same chat repeatedly
	chatKey := chat
	afkMu.Lock()
	if last, ok := afkLastReply[chatKey]; ok && time.Since(last) < afkCooldown {
		afkMu.Unlock()
		return
	}
	afkLastReply[chatKey] = time.Now()
	afkMu.Unlock()

	elapsed := time.Since(since).Round(time.Minute)
	minutes := int(elapsed.Minutes())
	timeStr := fmt.Sprintf("%d minute(s)", minutes)
	if minutes == 0 {
		timeStr = "just now"
	}

	msg := fmt.Sprintf(
		"🌙 *Owner is AFK* (Away From Keyboard)\n\n"+
			"⏱ Since: *%s*\n"+
			"📝 Reason: *%s*\n\n"+
			"_I'll reply when I'm back!_",
		timeStr, reason,
	)

	afkMsg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(msg),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:    proto.String(v.Info.ID),
				Participant: proto.String(v.Info.Sender.String()),
				QuotedMessage: &waProto.Message{
					Conversation: proto.String(extractBodyText(v)),
				},
			},
		},
	}

	ctx, cancel := newContext(10)
	defer cancel()
	_, _ = wac.SendMessage(ctx, v.Info.Chat, afkMsg)
}

// HandleAFK is an alias kept for compatibility with module.go hook calls
func HandleAFK(v *events.Message) {
	CheckAFK(v)
}

func afkHandler(v *events.Message, args []string) error {
	reason := "No reason given"
	if len(args) > 0 {
		reason = strings.Join(args, " ")
	}

	afkMu.Lock()
	afkEnabled = true
	afkReason = reason
	afkSince = time.Now()
	afkMu.Unlock()

	return ReplyErr(v, fmt.Sprintf(
		"🌙 *AFK enabled*\n📝 Reason: *%s*\n\nI'll auto-reply to anyone who messages or mentions me.",
		reason,
	))
}

func backHandler(v *events.Message, args []string) error {
	afkMu.Lock()
	wasAFK := afkEnabled
	since := afkSince
	afkEnabled = false
	afkReason = ""
	afkLastReply = make(map[string]time.Time) // reset cooldown on .back
	afkMu.Unlock()

	if !wasAFK {
		return ReplyErr(v, "ℹ️ You were not AFK.")
	}

	elapsed := time.Since(since).Round(time.Minute)
	return ReplyErr(v, fmt.Sprintf("✅ *Welcome back!*\nYou were AFK for *%s*.", elapsed))
}

func init() {
	Register(ModuleInfo{
		Name:        "AFK",
		Description: "Auto-reply when you are away from keyboard (.afk / .back)",
		Commands: []CommandInfo{
			{Pattern: "afk", Handler: afkHandler, Sudo: true},
			{Pattern: "back", Handler: backHandler, Sudo: true},
		},
	})
}
