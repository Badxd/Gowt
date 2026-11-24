package modules

import (
	"fmt"
	"strings"
	"sync"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// Welcome / Goodbye messages triggered on group participant join/leave events.
// .setwelcome <message>   — set join message for this group
// .setgoodbye <message>   — set leave message for this group
// .delwelcome             — remove welcome message
// .delgoodbye             — remove goodbye message
// .welcome                — show current welcome message
// .goodbye                — show current goodbye message
//
// Placeholders: {name} = user phone, {group} = group name

var (
	greetMu    sync.RWMutex
	welcomeMap = make(map[string]string) // chatJID -> welcome template
	goodbyeMap = make(map[string]string) // chatJID -> goodbye template
)

// HandleGroupJoin — called from HandleGroupParticipant in module.go
func HandleGroupJoin(evt *events.GroupInfo) {
	if evt == nil {
		return
	}
	chat := evt.JID.String()

	greetMu.RLock()
	tmpl, ok := welcomeMap[chat]
	greetMu.RUnlock()
	if !ok || tmpl == "" {
		return
	}

	wac := getClientSafe()
	if wac == nil {
		return
	}

	for _, jid := range evt.Join {
		msg := buildGreeting(tmpl, jid, evt.JID)
		ctx, cancel := newContext(10)
		_, _ = wac.SendMessage(ctx, evt.JID, &waProto.Message{Conversation: proto.String(msg)})
		cancel()
	}
}

// HandleGroupLeave — called from HandleGroupParticipant in module.go
func HandleGroupLeave(evt *events.GroupInfo) {
	if evt == nil {
		return
	}
	chat := evt.JID.String()

	greetMu.RLock()
	tmpl, ok := goodbyeMap[chat]
	greetMu.RUnlock()
	if !ok || tmpl == "" {
		return
	}

	wac := getClientSafe()
	if wac == nil {
		return
	}

	for _, jid := range evt.Leave {
		msg := buildGreeting(tmpl, jid, evt.JID)
		ctx, cancel := newContext(10)
		_, _ = wac.SendMessage(ctx, evt.JID, &waProto.Message{Conversation: proto.String(msg)})
		cancel()
	}
}

func buildGreeting(tmpl string, userJID types.JID, groupJID types.JID) string {
	name := "+" + userJID.User
	group := groupJID.User
	msg := strings.ReplaceAll(tmpl, "{name}", name)
	msg = strings.ReplaceAll(msg, "{group}", group)
	return msg
}

// ── Commands ──────────────────────────────────────────────────────────────────

func setWelcomeHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v,
			"📝 Usage: `.setwelcome <message>`\n"+
				"Placeholders: `{name}` = phone number, `{group}` = group name\n"+
				"Example: `.setwelcome Welcome {name} to {group}! 🎉`")
	}
	msg := strings.Join(args, " ")
	chat := v.Info.Chat.String()

	greetMu.Lock()
	welcomeMap[chat] = msg
	greetMu.Unlock()

	return ReplyErr(v, fmt.Sprintf("✅ Welcome message set:\n%s", msg))
}

func setGoodbyeHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v,
			"📝 Usage: `.setgoodbye <message>`\n"+
				"Placeholders: `{name}` = phone number, `{group}` = group name\n"+
				"Example: `.setgoodbye Goodbye {name}! We'll miss you 👋`")
	}
	msg := strings.Join(args, " ")
	chat := v.Info.Chat.String()

	greetMu.Lock()
	goodbyeMap[chat] = msg
	greetMu.Unlock()

	return ReplyErr(v, fmt.Sprintf("✅ Goodbye message set:\n%s", msg))
}

func delWelcomeHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()
	greetMu.Lock()
	delete(welcomeMap, chat)
	greetMu.Unlock()
	return ReplyErr(v, "✅ Welcome message removed.")
}

func delGoodbyeHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()
	greetMu.Lock()
	delete(goodbyeMap, chat)
	greetMu.Unlock()
	return ReplyErr(v, "✅ Goodbye message removed.")
}

func showWelcomeHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()
	greetMu.RLock()
	msg, ok := welcomeMap[chat]
	greetMu.RUnlock()
	if !ok || msg == "" {
		return ReplyErr(v, "ℹ️ No welcome message set. Use `.setwelcome <message>`")
	}
	return ReplyErr(v, fmt.Sprintf("👋 *Current welcome message:*\n%s", msg))
}

func showGoodbyeHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()
	greetMu.RLock()
	msg, ok := goodbyeMap[chat]
	greetMu.RUnlock()
	if !ok || msg == "" {
		return ReplyErr(v, "ℹ️ No goodbye message set. Use `.setgoodbye <message>`")
	}
	return ReplyErr(v, fmt.Sprintf("👋 *Current goodbye message:*\n%s", msg))
}

func init() {
	Register(ModuleInfo{
		Name:        "Welcome/Goodbye",
		Description: "Auto-send custom join/leave messages in groups",
		Commands: []CommandInfo{
			{Pattern: "setwelcome", Handler: setWelcomeHandler, Sudo: true},
			{Pattern: "setgoodbye", Handler: setGoodbyeHandler, Sudo: true},
			{Pattern: "delwelcome", Handler: delWelcomeHandler, Sudo: true},
			{Pattern: "delgoodbye", Handler: delGoodbyeHandler, Sudo: true},
			{Pattern: "welcome", Handler: showWelcomeHandler, Sudo: true},
			{Pattern: "goodbye", Handler: showGoodbyeHandler, Sudo: true},
		},
	})
}
