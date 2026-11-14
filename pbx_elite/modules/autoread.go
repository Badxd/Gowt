package modules

import (
	"context"
	"fmt"
	"sync"
	"time"


	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// .autoread   — toggle auto-read (mark messages as read automatically)
// .readall    — mark ALL pending messages in current chat as read right now

var (
	autoReadMu      sync.RWMutex
	autoReadEnabled bool // global toggle
)

// MarkAsRead is exported so the global handler can call it when autoread is on.
func MarkAsRead(v *events.Message) {
	autoReadMu.RLock()
	enabled := autoReadEnabled
	autoReadMu.RUnlock()

	if !enabled {
		return
	}

	wac := getClientSafe()
	if wac == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	msgIDs := []types.MessageID{v.Info.ID}
	_ = wac.MarkRead(ctx, msgIDs, v.Info.Timestamp, v.Info.Chat, v.Info.Sender)
}

func autoReadToggleHandler(v *events.Message, args []string) error {
	autoReadMu.Lock()
	autoReadEnabled = !autoReadEnabled
	state := autoReadEnabled
	autoReadMu.Unlock()

	if state {
		return ReplyErr(v, "✅ *Auto-Read ON* — all incoming messages will be marked as read.")
	}
	return ReplyErr(v, "🔕 *Auto-Read OFF*")
}

func readAllHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msgIDs := []types.MessageID{v.Info.ID}
	if err := wac.MarkRead(ctx, msgIDs, v.Info.Timestamp, v.Info.Chat, v.Info.Sender); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to mark as read: %v", err))
	}
	return ReplyErr(v, "✅ Marked as read.")
}

func init() {
	Register(ModuleInfo{
		Name:        "Auto Read",
		Description: "Automatically mark incoming messages as read",
		Commands: []CommandInfo{
			{Pattern: "autoread", Handler: autoReadToggleHandler, Sudo: true},
			{Pattern: "readall", Handler: readAllHandler, Sudo: true},
		},
	})
}
