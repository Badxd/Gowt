package modules

// ── REPLY RAID MODULE ────────────────────────────────────────────────────────
// Ports: replyraid.py, preplyraid.py, ereplyraid.py, hreplyraid.py, shreplyraid.py
//
// Architecture:
//   - Per-chat per-user target list (map[chatJID]map[userJID]poolName)
//   - CheckReplyRaid(v) is called from HandleMessage() on EVERY incoming message
//   - If sender is in target list for that chat, reply with random string from pool

import (
	"fmt"
	"strings"
	"sync"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

func init() {
	Register(ModuleInfo{
		Name:        "ReplyRaid",
		Description: "Auto-reply on every message from a target user with raid strings",
		Commands: []CommandInfo{
			{Pattern: "replyraid",    Handler: replyRaidAddHandler,   Sudo: true},
			{Pattern: "dreplyraid",   Handler: replyRaidDelHandler,   Sudo: true},
			{Pattern: "preplyraid",   Handler: preplyRaidAddHandler,  Sudo: true},
			{Pattern: "dpreplyraid",  Handler: preplyRaidDelHandler,  Sudo: true},
			{Pattern: "ereplyraid",   Handler: ereplyRaidAddHandler,  Sudo: true},
			{Pattern: "dereplyraid",  Handler: ereplyRaidDelHandler,  Sudo: true},
			{Pattern: "hreplyraid",   Handler: hreplyRaidAddHandler,  Sudo: true},
			{Pattern: "dhreplyraid",  Handler: hreplyRaidDelHandler,  Sudo: true},
			{Pattern: "shreplyraid",  Handler: shreplyRaidAddHandler, Sudo: true},
			{Pattern: "dshreplyraid", Handler: shreplyRaidDelHandler, Sudo: true},
			{Pattern: "listreplyraid",Handler: listReplyRaidHandler,  Sudo: true},
		},
	})
}

type replyRaidTarget struct {
	poolName string
}

var (
	replyRaidMu    sync.RWMutex
	replyRaidStore = make(map[string]map[string]replyRaidTarget)
)

func replyRaidPoolByName(poolName string) []string {
	switch poolName {
	case "replyraid":
		return REPLYRAID_POOL
	case "preplyraid":
		return PREPLYRAID_POOL
	case "ereplyraid":
		return EREPLYRAID_POOL
	case "hreplyraid":
		return HREPLYRAID_POOL
	case "shreplyraid":
		return SHAYRI_POOL
	default:
		return REPLYRAID_POOL
	}
}

// CheckReplyRaid is called from HandleMessage on every incoming message.
func CheckReplyRaid(v *events.Message) {
	wac := getClientSafe()
	if wac == nil {
		return
	}
	if v.Info.IsFromMe {
		return
	}

	chatKey := v.Info.Chat.String()
	senderKey := v.Info.Sender.ToNonAD().String()

	replyRaidMu.RLock()
	chatMap, ok := replyRaidStore[chatKey]
	if !ok {
		replyRaidMu.RUnlock()
		return
	}
	target, ok := chatMap[senderKey]
	replyRaidMu.RUnlock()
	if !ok {
		return
	}

	pool := replyRaidPoolByName(target.poolName)
	msg := "@" + v.Info.Sender.ToNonAD().User + " " + randItem(pool)

	senderJIDStr := v.Info.Sender.String()

	protoMsg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: strPtr(msg),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID:  []string{v.Info.Sender.ToNonAD().String()},
				StanzaID:      strPtr(v.Info.ID),
				Participant:   strPtr(senderJIDStr),
				QuotedMessage: v.Message,
			},
		},
	}

	ctx, cancel := newContext(10)
	_, _ = wac.SendMessage(ctx, v.Info.Chat, protoMsg)
	cancel()
}

// ── ADD / REMOVE ─────────────────────────────────────────────────────────────

func replyRaidAdd(v *events.Message, args []string, poolName string) error {
	if len(args) < 1 {
		safeSendMessage(v, fmt.Sprintf("Usage: .%s <phone>", poolName))
		return nil
	}

	jid, _ := resolveMentionOrPhone(v, args[0])
	if jid.IsEmpty() {
		safeSendMessage(v, "Could not resolve user: "+args[0])
		return nil
	}

	chatKey := v.Info.Chat.String()
	senderKey := jid.ToNonAD().String()

	replyRaidMu.Lock()
	if replyRaidStore[chatKey] == nil {
		replyRaidStore[chatKey] = make(map[string]replyRaidTarget)
	}
	replyRaidStore[chatKey][senderKey] = replyRaidTarget{poolName: poolName}
	replyRaidMu.Unlock()

	safeSendMessage(v, fmt.Sprintf("✅ Reply raid enabled for +%s [pool: %s]", jid.User, poolName))
	return nil
}

func replyRaidRemove(v *events.Message, args []string) error {
	if len(args) < 1 {
		safeSendMessage(v, "Usage: .dreplyraid <phone>")
		return nil
	}

	jid, _ := resolveMentionOrPhone(v, args[0])
	if jid.IsEmpty() {
		safeSendMessage(v, "Could not resolve user: "+args[0])
		return nil
	}

	chatKey := v.Info.Chat.String()
	senderKey := jid.ToNonAD().String()

	replyRaidMu.Lock()
	if replyRaidStore[chatKey] != nil {
		delete(replyRaidStore[chatKey], senderKey)
		if len(replyRaidStore[chatKey]) == 0 {
			delete(replyRaidStore, chatKey)
		}
	}
	replyRaidMu.Unlock()

	safeSendMessage(v, fmt.Sprintf("✅ Reply raid removed for +%s", jid.User))
	return nil
}

// ── HANDLER WRAPPERS ─────────────────────────────────────────────────────────

func replyRaidAddHandler(v *events.Message, args []string) error {
	return replyRaidAdd(v, args, "replyraid")
}
func replyRaidDelHandler(v *events.Message, args []string) error {
	return replyRaidRemove(v, args)
}
func preplyRaidAddHandler(v *events.Message, args []string) error {
	return replyRaidAdd(v, args, "preplyraid")
}
func preplyRaidDelHandler(v *events.Message, args []string) error {
	return replyRaidRemove(v, args)
}
func ereplyRaidAddHandler(v *events.Message, args []string) error {
	return replyRaidAdd(v, args, "ereplyraid")
}
func ereplyRaidDelHandler(v *events.Message, args []string) error {
	return replyRaidRemove(v, args)
}
func hreplyRaidAddHandler(v *events.Message, args []string) error {
	return replyRaidAdd(v, args, "hreplyraid")
}
func hreplyRaidDelHandler(v *events.Message, args []string) error {
	return replyRaidRemove(v, args)
}
func shreplyRaidAddHandler(v *events.Message, args []string) error {
	return replyRaidAdd(v, args, "shreplyraid")
}
func shreplyRaidDelHandler(v *events.Message, args []string) error {
	return replyRaidRemove(v, args)
}

func listReplyRaidHandler(v *events.Message, args []string) error {
	chatKey := v.Info.Chat.String()

	replyRaidMu.RLock()
	chatMap := replyRaidStore[chatKey]
	replyRaidMu.RUnlock()

	if len(chatMap) == 0 {
		safeSendMessage(v, "No active reply raids in this chat.")
		return nil
	}

	var sb strings.Builder
	sb.WriteString("*Active Reply Raids:*\n")
	for userKey, target := range chatMap {
		sb.WriteString(fmt.Sprintf("• +%s → [%s]\n", userKey, target.poolName))
	}
	safeSendMessage(v, sb.String())
	return nil
}
