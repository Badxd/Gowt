package modules

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// PBX Elite — Shayari Module
// ✅ .shayari <count>
// ✅ .stopshayari

var (
	activeShayariChats   = make(map[string]bool)
	activeShayariChatsMu sync.Mutex
)

func setShayariActive(chatID string) {
	activeShayariChatsMu.Lock()
	activeShayariChats[chatID] = true
	activeShayariChatsMu.Unlock()
}

func setShayariStopped(chatID string) {
	activeShayariChatsMu.Lock()
	delete(activeShayariChats, chatID)
	activeShayariChatsMu.Unlock()
}

func isShayariActive(chatID string) bool {
	activeShayariChatsMu.Lock()
	defer activeShayariChatsMu.Unlock()
	return activeShayariChats[chatID]
}

// ─────────────────────────────────────────────
// .shayari <count>
// ─────────────────────────────────────────────
func shayariHandler(v *events.Message, args []string) error {
	var count int
	fmt.Sscanf(GetArgs(v), "%d", &count)

	if count < 1 {
		count = 5
	}
	if count > 100 {
		count = 100
	}

	chatID := v.Info.Chat.String()

	if isShayariActive(chatID) {
		return ReplyErr(v, "⚠️ Shayari already running. Use `.stopshayari`")
	}

	list := ShayariList
	if len(list) == 0 {
		return ReplyErr(v, "❌ Shayari list is empty in config.")
	}

	revokeCmd(v) // Delete command message

	setShayariActive(chatID)
	defer setShayariStopped(chatID)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < count; i++ {
		if !isShayariActive(chatID) {
			break
		}

		shayari := list[rng.Intn(len(list))]

		if err := ReplyErr(v, shayari); err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		time.Sleep(700 * time.Millisecond)
	}

	return nil
}

// ─────────────────────────────────────────────
// .stopshayari
// ─────────────────────────────────────────────
func stopShayariHandler(v *events.Message, args []string) error {
	chatID := v.Info.Chat.String()

	if !isShayariActive(chatID) {
		return ReplyErr(v, "ℹ️ No active shayari session.")
	}

	setShayariStopped(chatID)
	revokeCmd(v)

	return ReplyErr(v, "🛑 *Shayari Stopped!*")
}

// REGISTER
func init() {
	Register(ModuleInfo{
		Name:        "Shayari",
		Description: "Random Shayari Sender",

		Commands: []CommandInfo{
			{Pattern: "shayari", Handler: shayariHandler, Sudo: true},
			{Pattern: "stopshayari", Handler: stopShayariHandler, Sudo: true},
		},
	})
}
