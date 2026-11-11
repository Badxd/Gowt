package modules

import (
	"fmt"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// Welcome / Goodbye messages triggered on group participant join/leave events.
// Stored per chatJID. Supports {name} and {group} placeholders.
//
// .setwelcome <message>   — set join message for this group
// .setgoodbye <message>   — set leave message for this group
// .delwelcome            — remove welcome message
// .delgoodbye            — remove goodbye message
// .welcome               — show current welcome message
// .goodbye               — show current goodbye message

var (
	greetMu      sync.RWMutex
	welcomeMap   = make(map[string]string) // chatJID -> welcome template
	goodbyeMap   = make(map[string]string) // chatJID -> goodbye template
)

// HandleGroupJoin is the exported hook — called from the group participant event handler.
// Wire this into main.go / bot.go where *events.GroupInfo is handled.
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

	// evt.Join is []types.JID of users who joined
	for _, jid := range evt.Join {
		msg := buildGreeting(tmpl, jid, evt.JID)
		SendText(wac, evt.JID, msg)
	}
}

// HandleGroupLeave is the exported hook — called for leave events.
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
		SendText(wac, evt.JID, msg)
	}
}

// buildGreeting replaces {name} and {group} placeholders.
func buildGreeting(tmpl string, userJID types.JID, groupJID types.JID) string {
	name := "+" + userJID.User
	group := groupJID.User

	msg := strings.ReplaceAll(tmpl, "{name}", name)
	msg = strings.ReplaceAll(msg, "{group}", group)
	return msg
}





func showWelcomeHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()
	greetMu.RLock()
	msg, ok := welcomeMap[chat]
	greetMu.RUnlock()
	if !ok || msg == "" {
		return ReplyErr(v, "ℹ️ No welcome message set. Use .setwelcome <message>")
	}
	return ReplyErr(v, fmt.Sprintf("👋 *Current welcome message:*\n%s", msg))
}

func showGoodbyeHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()
	greetMu.RLock()
	msg, ok := goodbyeMap[chat]
	greetMu.RUnlock()
	if !ok || msg == "" {
		return ReplyErr(v, "ℹ️ No goodbye message set. Use .setgoodbye <message>")
	}
	return ReplyErr(v, fmt.Sprintf("👋 *Current goodbye message:*\n%s", msg))
}

func init() {
	Register(ModuleInfo{
		Name:        "Welcome/Goodbye",
		Description: "Auto-send custom join/leave messages in groups",
		Commands: []CommandInfo{
			{Pattern: "welcome", Handler: showWelcomeHandler, Sudo: true},
			{Pattern: "goodbye", Handler: showGoodbyeHandler, Sudo: true},
		},
	})
}
