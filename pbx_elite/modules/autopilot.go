package modules

import (
	"fmt"
	"strings"
	"sync"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// Autopilot / auto-reply: auto-respond to messages when enabled.
//
// .autopilot on/off/status     — toggle autopilot for this chat
// .setreply <trigger> | <reply> — set a trigger→reply pair
// .delreply <trigger>           — remove a trigger
// .autoreply                    — list all auto-replies for this chat

type autoReplyEntry struct {
	Trigger string
	Reply   string
}

var (
	autopilotMu      sync.RWMutex
	autopilotEnabled = make(map[string]bool)             // chatJID -> enabled
	autoReplies      = make(map[string][]*autoReplyEntry) // chatJID -> entries
)

// CheckAutoPilot is the exported hook — sends auto-replies when triggered.
func CheckAutoPilot(v *events.Message) {
	chat := v.Info.Chat.String()

	autopilotMu.RLock()
	enabled := autopilotEnabled[chat]
	entries := autoReplies[chat]
	autopilotMu.RUnlock()

	if !enabled || len(entries) == 0 {
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

	body := strings.ToLower(strings.TrimSpace(extractBodyText(v)))
	if body == "" {
		return
	}

	for _, e := range entries {
		if strings.Contains(body, strings.ToLower(e.Trigger)) {
			ctx, cancel := newContext(8)
			msg := &waProto.Message{Conversation: proto.String(e.Reply)}
			_, _ = wac.SendMessage(ctx, v.Info.Chat, msg)
			cancel()
			return // only first match fires
		}
	}
}

func autopilotHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()

	if len(args) == 0 {
		autopilotMu.RLock()
		en := autopilotEnabled[chat]
		count := len(autoReplies[chat])
		autopilotMu.RUnlock()
		status := "🔴 disabled"
		if en {
			status = "🟢 enabled"
		}
		return ReplyErr(v, fmt.Sprintf("🤖 Autopilot: *%s*\n📋 Auto-replies: *%d*\n\nUse .autopilot on/off", status, count))
	}

	switch strings.ToLower(args[0]) {
	case "on", "enable":
		autopilotMu.Lock()
		autopilotEnabled[chat] = true
		autopilotMu.Unlock()
		return ReplyErr(v, "✅ Autopilot *enabled*. Bot will auto-reply to triggers.")
	case "off", "disable":
		autopilotMu.Lock()
		autopilotEnabled[chat] = false
		autopilotMu.Unlock()
		return ReplyErr(v, "✅ Autopilot *disabled*.")
	case "status":
		autopilotMu.RLock()
		en := autopilotEnabled[chat]
		autopilotMu.RUnlock()
		status := "🔴 disabled"
		if en {
			status = "🟢 enabled"
		}
		return ReplyErr(v, fmt.Sprintf("🤖 Autopilot: *%s*", status))
	default:
		return ReplyErr(v, "Usage: .autopilot on/off/status")
	}
}

func setReplyHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .setreply <trigger> | <reply>\nExample: .setreply hello | Hey there! 👋")
	}

	full := strings.Join(args, " ")
	parts := strings.SplitN(full, "|", 2)
	if len(parts) != 2 {
		return ReplyErr(v, "Usage: .setreply <trigger> | <reply>\nSeparate trigger and reply with |")
	}

	trigger := strings.TrimSpace(parts[0])
	reply := strings.TrimSpace(parts[1])

	if trigger == "" || reply == "" {
		return ReplyErr(v, "❌ Trigger and reply cannot be empty.")
	}

	chat := v.Info.Chat.String()

	autopilotMu.Lock()
	// update if exists
	for _, e := range autoReplies[chat] {
		if strings.ToLower(e.Trigger) == strings.ToLower(trigger) {
			e.Reply = reply
			autopilotMu.Unlock()
			return ReplyErr(v, fmt.Sprintf("✅ Auto-reply for *%s* updated.", trigger))
		}
	}
	autoReplies[chat] = append(autoReplies[chat], &autoReplyEntry{
		Trigger: trigger,
		Reply:   reply,
	})
	autopilotMu.Unlock()

	return ReplyErr(v, fmt.Sprintf("✅ Auto-reply set.\n🔑 Trigger: *%s*\n💬 Reply: %s", trigger, reply))
}

func delReplyHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .delreply <trigger>")
	}

	trigger := strings.ToLower(strings.Join(args, " "))
	chat := v.Info.Chat.String()

	autopilotMu.Lock()
	list := autoReplies[chat]
	newList := list[:0]
	found := false
	for _, e := range list {
		if strings.ToLower(e.Trigger) == trigger {
			found = true
			continue
		}
		newList = append(newList, e)
	}
	autoReplies[chat] = newList
	autopilotMu.Unlock()

	if !found {
		return ReplyErr(v, fmt.Sprintf("❌ No auto-reply found for trigger *%s*.", trigger))
	}
	return ReplyErr(v, fmt.Sprintf("✅ Auto-reply for *%s* deleted.", trigger))
}

func autoReplyListHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()

	autopilotMu.RLock()
	list := autoReplies[chat]
	enabled := autopilotEnabled[chat]
	autopilotMu.RUnlock()

	if len(list) == 0 {
		return ReplyErr(v, "ℹ️ No auto-replies set. Use .setreply trigger | response to add one.")
	}

	status := "🔴 off"
	if enabled {
		status = "🟢 on"
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("🤖 *Auto-replies (%s) — %d:*", status, len(list)), "")
	for i, e := range list {
		preview := e.Reply
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		lines = append(lines, fmt.Sprintf("%d. *%s* → %s", i+1, e.Trigger, preview))
	}
	return ReplyErr(v, strings.Join(lines, "\n"))
}

func init() {
	Register(ModuleInfo{
		Name:        "Autopilot",
		Description: "Auto-reply to trigger keywords when autopilot is enabled",
		Commands: []CommandInfo{
			{Pattern: "autopilot", Handler: autopilotHandler, Sudo: true},
			{Pattern: "setreply", Handler: setReplyHandler, Sudo: true},
			{Pattern: "delreply", Handler: delReplyHandler, Sudo: true},
			{Pattern: "listreplies", Handler: autoReplyListHandler, Sudo: true},
		},
	})
}
