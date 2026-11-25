package modules

import (
	"fmt"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow/types/events"
)

// .setnick @user <nickname>   — assign a nickname to a user in this chat
// .nick @user                 — show a user's nickname
// .delnick @user              — remove a user's nickname
// .nicklist                   — list all nicknames in this chat

var (
	nickMu   sync.RWMutex
	nickMap  = make(map[string]string) // "chatJID|userJID" -> nickname
)

func setNickHandler(v *events.Message, args []string) error {
	targetJID, err := extractTargetJID(v, args)
	if err != nil || len(args) < 2 {
		return ReplyErr(v, "Usage: .setnick @user <nickname>")
	}

	// nickname is everything after the first arg (which was the @mention or number)
	// extractTargetJID consumed args[0] if it was a phone number
	nick := ""
	if len(args) >= 2 {
		nick = strings.Join(args[1:], " ")
	}
	if nick == "" {
		return ReplyErr(v, "❌ Nickname cannot be empty.")
	}

	key := v.Info.Chat.String() + "|" + targetJID

	nickMu.Lock()
	nickMap[key] = nick
	nickMu.Unlock()

	user := strings.SplitN(targetJID, "@", 2)[0]
	return ReplyErr(v, fmt.Sprintf("✅ Nickname for @%s set to *%s*.", user, nick))
}

func getNickHandler(v *events.Message, args []string) error {
	targetJID, err := extractTargetJID(v, args)
	if err != nil {
		return ReplyErr(v, "Usage: .nick @user (or reply to their message)")
	}

	key := v.Info.Chat.String() + "|" + targetJID

	nickMu.RLock()
	nick, ok := nickMap[key]
	nickMu.RUnlock()

	user := strings.SplitN(targetJID, "@", 2)[0]
	if !ok || nick == "" {
		return ReplyErr(v, fmt.Sprintf("ℹ️ No nickname set for @%s.", user))
	}

	return ReplyErr(v, fmt.Sprintf("🏷️ @%s → *%s*", user, nick))
}

func delNickHandler(v *events.Message, args []string) error {
	targetJID, err := extractTargetJID(v, args)
	if err != nil {
		return ReplyErr(v, "Usage: .delnick @user (or reply to their message)")
	}

	key := v.Info.Chat.String() + "|" + targetJID

	nickMu.Lock()
	_, had := nickMap[key]
	delete(nickMap, key)
	nickMu.Unlock()

	user := strings.SplitN(targetJID, "@", 2)[0]
	if !had {
		return ReplyErr(v, fmt.Sprintf("ℹ️ No nickname was set for @%s.", user))
	}

	return ReplyErr(v, fmt.Sprintf("🗑️ Removed nickname for @%s.", user))
}

func nickListHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()
	prefix := chat + "|"

	nickMu.RLock()
	var lines []string
	for k, nick := range nickMap {
		if strings.HasPrefix(k, prefix) {
			userPart := strings.TrimPrefix(k, prefix)
			jidParts := strings.SplitN(userPart, "@", 2)
			user := jidParts[0]
			lines = append(lines, fmt.Sprintf("• @%s → *%s*", user, nick))
		}
	}
	nickMu.RUnlock()

	if len(lines) == 0 {
		return ReplyErr(v, "📋 No nicknames set in this chat.")
	}

	return ReplyErr(v, "🏷️ *Nicknames in this chat:*\n\n"+strings.Join(lines, "\n"))
}

func init() {
	Register(ModuleInfo{
		Name:        "Nicknames",
		Description: "Assign and manage per-user nicknames",
		Commands: []CommandInfo{
			{Pattern: "setnick", Handler: setNickHandler, Sudo: true},
			{Pattern: "nick", Handler: getNickHandler, Sudo: true},
			{Pattern: "delnick", Handler: delNickHandler, Sudo: true},
			{Pattern: "nicklist", Handler: nickListHandler, Sudo: true},
		},
	})
}
