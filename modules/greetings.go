package modules

import (
	"fmt"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow/types/events"
)

// greetingStore holds welcome/goodbye messages per group JID
type greetingEntry struct {
	Welcome string
	Goodbye string
}

var (
	greetingMu    sync.RWMutex
	greetingStore = make(map[string]*greetingEntry)
)

func getGreeting(jid string) *greetingEntry {
	greetingMu.RLock()
	defer greetingMu.RUnlock()
	if e, ok := greetingStore[jid]; ok {
		return e
	}
	return nil
}

func setWelcomeMsg(jid, msg string) {
	greetingMu.Lock()
	defer greetingMu.Unlock()
	if _, ok := greetingStore[jid]; !ok {
		greetingStore[jid] = &greetingEntry{}
	}
	greetingStore[jid].Welcome = msg
}

func setGoodbyeMsg(jid, msg string) {
	greetingMu.Lock()
	defer greetingMu.Unlock()
	if _, ok := greetingStore[jid]; !ok {
		greetingStore[jid] = &greetingEntry{}
	}
	greetingStore[jid].Goodbye = msg
}

func delWelcome(jid string) {
	greetingMu.Lock()
	defer greetingMu.Unlock()
	if e, ok := greetingStore[jid]; ok {
		e.Welcome = ""
	}
}

func delGoodbye(jid string) {
	greetingMu.Lock()
	defer greetingMu.Unlock()
	if e, ok := greetingStore[jid]; ok {
		e.Goodbye = ""
	}
}

// --- Command Handlers ---

func setWelcomeHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .setwelcome <message>\nYou can use {name} for the new member's name.")
	}
	jid := v.Info.Chat.String()
	msg := strings.Join(args, " ")
	setWelcomeMsg(jid, msg)
	return ReplyErr(v, fmt.Sprintf("✅ Welcome message set:\n\n%s", msg))
}

func setGoodbyeHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .setgoodbye <message>\nYou can use {name} for the leaving member's name.")
	}
	jid := v.Info.Chat.String()
	msg := strings.Join(args, " ")
	setGoodbyeMsg(jid, msg)
	return ReplyErr(v, fmt.Sprintf("✅ Goodbye message set:\n\n%s", msg))
}

func delWelcomeHandler(v *events.Message, args []string) error {
	jid := v.Info.Chat.String()
	delWelcome(jid)
	return ReplyErr(v, "✅ Welcome message deleted.")
}

func delGoodbyeHandler(v *events.Message, args []string) error {
	jid := v.Info.Chat.String()
	delGoodbye(jid)
	return ReplyErr(v, "✅ Goodbye message deleted.")
}

func greetingInfoHandler(v *events.Message, args []string) error {
	jid := v.Info.Chat.String()
	e := getGreeting(jid)
	if e == nil || (e.Welcome == "" && e.Goodbye == "") {
		return ReplyErr(v, "ℹ️ No greetings set for this chat.")
	}
	msg := "🎉 *Greetings for this chat:*\n\n"
	if e.Welcome != "" {
		msg += "👋 *Welcome:*\n" + e.Welcome + "\n\n"
	} else {
		msg += "👋 *Welcome:* Not set\n\n"
	}
	if e.Goodbye != "" {
		msg += "👋 *Goodbye:*\n" + e.Goodbye
	} else {
		msg += "👋 *Goodbye:* Not set"
	}
	return ReplyErr(v, msg)
}

func init() {
	Register(ModuleInfo{
		Name:        "Greetings",
		Description: "Group welcome/goodbye message management",
		Commands: []CommandInfo{
			{Pattern: "setwelcome", Handler: setWelcomeHandler, Sudo: true},
			{Pattern: "setgoodbye", Handler: setGoodbyeHandler, Sudo: true},
			{Pattern: "delwelcome", Handler: delWelcomeHandler, Sudo: true},
			{Pattern: "delgoodbye", Handler: delGoodbyeHandler, Sudo: true},
			{Pattern: "greetinfo", Handler: greetingInfoHandler, Sudo: true},
		},
	})
}
