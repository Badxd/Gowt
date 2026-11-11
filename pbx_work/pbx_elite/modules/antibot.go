package modules

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"PbxGb/database"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// Antibot: auto-kick users whose messages look like bot activity
// .antibot on/off  — toggle antibot for this group
// .antibotlist     — show groups with antibot enabled

var (
	antibotMu      sync.RWMutex
	antibotEnabled = make(map[string]bool) // chatJID -> enabled
)

// heuristics for bot detection
var botPatterns = []string{
	"join our group",
	"earn money",
	"free followers",
	"subscribe now",
	"click the link",
	"t.me/",
	"bit.ly/",
	"tinyurl.com",
	"shorturl.at",
	"wa.me/",
	"wa.link/",
}

// CheckAntibot is exported hook called from HandleMessage
func CheckAntibot(v *events.Message) {
	chat := v.Info.Chat.String()
	if !strings.Contains(chat, "@g.us") {
		return
	}

	antibotMu.RLock()
	enabled := antibotEnabled[chat]
	antibotMu.RUnlock()

	if !enabled {
		return
	}

	// don't check own messages
	wac := getClientSafe()
	if wac == nil || wac.Store.ID == nil {
		return
	}
	if v.Info.Sender.String() == wac.Store.ID.String() {
		return
	}

	body := strings.ToLower(extractBodyText(v))
	if body == "" {
		return
	}

	matched := false
	for _, pat := range botPatterns {
		if strings.Contains(body, pat) {
			matched = true
			break
		}
	}

	if !matched {
		return
	}

	// kick the sender
	senderJID := v.Info.Sender
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	chatJID, err := types.ParseJID(chat)
	if err != nil {
		return
	}

	phone := senderJID.User

	// try to remove
	_, err = wac.UpdateGroupParticipants(ctx, chatJID, []types.JID{senderJID}, "remove")
	if err != nil {
		return
	}

	// notify
	_ = SendText(wac, chatJID, fmt.Sprintf("🤖 *Antibot:* Removed @%s for sending bot/spam content.", phone))
}

func antibotHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	if len(args) == 0 {
		chatJID := v.Info.Chat.String()
		antibotMu.RLock()
		state := antibotEnabled[chatJID]
		antibotMu.RUnlock()

		status := "disabled 🔴"
		if state {
			status = "enabled 🟢"
		}
		return ReplyErr(v, fmt.Sprintf("🤖 *Antibot* is currently *%s* in this group.", status))
	}

	chat := v.Info.Chat.String()
	switch strings.ToLower(args[0]) {
	case "on", "enable":
		antibotMu.Lock()
		antibotEnabled[chat] = true
		antibotMu.Unlock()
		_ = database.SetToggle("antibot", chat, true)
		return ReplyErr(v, "✅ *Antibot* enabled. Bot accounts/spammers will be auto-kicked.")
	case "off", "disable":
		antibotMu.Lock()
		delete(antibotEnabled, chat)
		antibotMu.Unlock()
		_ = database.DeleteToggle("antibot", chat)
		return ReplyErr(v, "🔴 *Antibot* disabled.")
	default:
		return ReplyErr(v, "Usage: .antibot on/off")
	}
}

func antibotListHandler(v *events.Message, args []string) error {
	antibotMu.RLock()
	defer antibotMu.RUnlock()

	if len(antibotEnabled) == 0 {
		return ReplyErr(v, "ℹ️ Antibot is not enabled in any group.")
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("🤖 *Antibot enabled in %d group(s):*", len(antibotEnabled)))
	i := 1
	for jid := range antibotEnabled {
		parts := strings.Split(jid, "@")
		lines = append(lines, fmt.Sprintf("  %d. `%s`", i, parts[0]))
		i++
	}

	return ReplyErr(v, strings.Join(lines, "\n"))
}

func init() {
	// Restore persisted antibot toggles from MongoDB.
	loaded := database.LoadAllToggles("antibot")
	antibotMu.Lock()
	for chatJID, enabled := range loaded {
		if enabled {
			antibotEnabled[chatJID] = true
		}
	}
	antibotMu.Unlock()

	Register(ModuleInfo{
		Name:        "Antibot",
		Description: "Auto-kick bot/spam accounts in groups",
		Commands: []CommandInfo{
			{Pattern: "antibot", Handler: antibotHandler, Sudo: true},
			{Pattern: "antibotlist", Handler: antibotListHandler, Sudo: true},
		},
	})
}
