package modules

import (
	"fmt"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow/types/events"
)

// .chatlock   — lock this chat (bot ignores all commands here)
// .chatunlock — unlock this chat
// .lockstatus — show lock status

var (
	chatLockMu      sync.RWMutex
	lockedChats     = make(map[string]bool) // chatJID -> locked
)

// IsChatLocked returns true if the chat has been locked via .chatlock.
// Called from module.go before command dispatch.
func IsChatLocked(chatJID string) bool {
	chatLockMu.RLock()
	defer chatLockMu.RUnlock()
	return lockedChats[chatJID]
}

func chatlockHandler(v *events.Message, args []string) error {
	chatJID := v.Info.Chat.String()

	chatLockMu.Lock()
	lockedChats[chatJID] = true
	chatLockMu.Unlock()

	return ReplyErr(v, "🔒 Chat locked. Bot will ignore commands in this chat.\nUse `.chatunlock` to re-enable.")
}

func chatunlockHandler(v *events.Message, args []string) error {
	chatJID := v.Info.Chat.String()

	chatLockMu.Lock()
	delete(lockedChats, chatJID)
	chatLockMu.Unlock()

	return ReplyErr(v, "🔓 Chat unlocked. Bot will respond to commands again.")
}

func lockStatusHandler(v *events.Message, args []string) error {
	chatJID := v.Info.Chat.String()

	chatLockMu.RLock()
	locked := lockedChats[chatJID]
	totalLocked := len(lockedChats)
	chatLockMu.RUnlock()

	status := "🔓 *Unlocked*"
	if locked {
		status = "🔒 *Locked*"
	}

	lines := []string{
		"🔐 *Chat Lock Status*",
		"",
		fmt.Sprintf("This chat: %s", status),
		fmt.Sprintf("Total locked chats: %d", totalLocked),
	}

	if totalLocked > 0 {
		chatLockMu.RLock()
		var lockedList []string
		for jid := range lockedChats {
			parts := strings.Split(jid, "@")
			if len(parts) > 0 {
				lockedList = append(lockedList, "  • "+parts[0])
			}
		}
		chatLockMu.RUnlock()
		if len(lockedList) > 0 {
			lines = append(lines, "", "*Locked chats:*")
			lines = append(lines, lockedList...)
		}
	}

	return ReplyErr(v, strings.Join(lines, "\n"))
}

func init() {
	Register(ModuleInfo{
		Name:        "Chat Lock",
		Description: "Lock/unlock a chat to ignore all commands in it",
		Commands: []CommandInfo{
			{Pattern: "chatlock", Handler: chatlockHandler, Sudo: true},
			{Pattern: "chatunlock", Handler: chatunlockHandler, Sudo: true},
			{Pattern: "lockstatus", Handler: lockStatusHandler, Sudo: true},
		},
	})
}
