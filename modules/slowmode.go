package modules

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// Slow mode: per-chat cooldown between messages from the same user
// .slowmode <seconds>  — enable slow mode with N second cooldown
// .slowmode off        — disable slow mode
// .slowmode            — show current status

var (
	slowModeMu      sync.RWMutex
	slowModeChats   = make(map[string]time.Duration) // chatJID -> cooldown duration
	slowModeTracker = make(map[string]time.Time)     // "chatJID|userJID" -> last message time
)

// CheckSlowMode is the exported hook — deletes messages that violate slow mode.
func CheckSlowMode(v *events.Message) {
	chat := v.Info.Chat.String()
	if !strings.Contains(chat, "@g.us") {
		return
	}

	slowModeMu.RLock()
	cooldown, enabled := slowModeChats[chat]
	slowModeMu.RUnlock()

	if !enabled || cooldown <= 0 {
		return
	}

	wac := getClientSafe()
	if wac == nil || wac.Store.ID == nil {
		return
	}

	sender := v.Info.Sender.String()
	// never apply to self
	if strings.HasPrefix(sender, wac.Store.ID.User) {
		return
	}

	key := chat + "|" + normalizeJIDForSudo(sender)

	slowModeMu.Lock()
	lastMsg, seen := slowModeTracker[key]
	now := time.Now()
	if !seen || now.Sub(lastMsg) >= cooldown {
		slowModeTracker[key] = now
		slowModeMu.Unlock()
		return
	}
	// within cooldown — delete the message
	slowModeMu.Unlock()

	// revoke
	ctx, cancel := newContext(8)
	defer cancel()
	_, _ = wac.RevokeMessage(ctx, v.Info.Chat, v.Info.ID)
}

func slowModeHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	chat := v.Info.Chat.String()

	if len(args) == 0 {
		slowModeMu.RLock()
		cd, active := slowModeChats[chat]
		slowModeMu.RUnlock()

		if !active {
			return ReplyErr(v, "🐢 *Slow mode* is currently *disabled*.")
		}
		return ReplyErr(v, fmt.Sprintf("🐢 *Slow mode* is *enabled*: %s cooldown between messages.", cd))
	}

	arg := strings.ToLower(args[0])

	if arg == "off" || arg == "disable" {
		slowModeMu.Lock()
		delete(slowModeChats, chat)
		slowModeMu.Unlock()
		return ReplyErr(v, "✅ *Slow mode* disabled.")
	}

	var secs int
	if _, err := fmt.Sscan(arg, &secs); err != nil || secs <= 0 {
		return ReplyErr(v, "Usage: .slowmode <seconds> or .slowmode off\nExample: .slowmode 10")
	}
	if secs > 3600 {
		secs = 3600 // max 1 hour
	}

	cooldown := time.Duration(secs) * time.Second

	slowModeMu.Lock()
	slowModeChats[chat] = cooldown
	slowModeMu.Unlock()

	return ReplyErr(v, fmt.Sprintf("🐢 *Slow mode* enabled: *%s* cooldown between messages per user.", cooldown))
}

func init() {
	Register(ModuleInfo{
		Name:        "Slow Mode",
		Description: "Per-chat cooldown between messages from the same user",
		Commands: []CommandInfo{
			{Pattern: "slowmode", Handler: slowModeHandler, Sudo: true},
		},
	})
}
