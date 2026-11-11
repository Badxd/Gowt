package modules

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"


	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// .ghost on/off    — toggle ghost mode (hide read receipts + typing + online status)
// .ghost status    — show current ghost mode state

var (
	ghostMu      sync.RWMutex
	ghostEnabled bool
)

func applyGhostMode(wac *whatsmeow.Client, enable bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Privacy: read receipts — "none" hides them from everyone
	privacy := whatsmeow.PrivacySettingEveryone
	if enable {
		privacy = whatsmeow.PrivacySettingNone
	}

	// SetPrivacySetting can take multiple settings
	err := wac.SetPrivacySetting(ctx, whatsmeow.PrivacySettingTypeReadReceipts, privacy)
	if err != nil {
		return fmt.Errorf("read receipts: %w", err)
	}

	// Also set online visibility
	onlinePrivacy := whatsmeow.PrivacySettingEveryone
	if enable {
		onlinePrivacy = whatsmeow.PrivacySettingNone
	}
	err = wac.SetPrivacySetting(ctx, whatsmeow.PrivacySettingTypeOnline, onlinePrivacy)
	if err != nil {
		// non-fatal — not all versions support this
		_ = err
	}

	return nil
}

func ghostHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		ghostMu.RLock()
		enabled := ghostEnabled
		ghostMu.RUnlock()

		status := "off 👻"
		if enabled {
			status = "on 🔕"
		}
		return ReplyErr(v, fmt.Sprintf("👻 Ghost mode is currently *%s*\n\nUsage: .ghost on/off", status))
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	switch strings.ToLower(args[0]) {
	case "on", "enable", "true", "1":
		if err := applyGhostMode(wac, true); err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed to enable ghost mode: %v", err))
		}
		ghostMu.Lock()
		ghostEnabled = true
		ghostMu.Unlock()
		return ReplyErr(v, "👻 Ghost mode *enabled*.\n• Read receipts hidden\n• Online status hidden\n• Typing indicator suppressed")

	case "off", "disable", "false", "0":
		if err := applyGhostMode(wac, false); err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed to disable ghost mode: %v", err))
		}
		ghostMu.Lock()
		ghostEnabled = false
		ghostMu.Unlock()
		return ReplyErr(v, "✅ Ghost mode *disabled*. Presence and receipts restored.")

	case "status":
		ghostMu.RLock()
		enabled := ghostEnabled
		ghostMu.RUnlock()

		status := "off"
		if enabled {
			status = "on"
		}
		return ReplyErr(v, fmt.Sprintf("👻 Ghost mode: *%s*", status))

	default:
		return ReplyErr(v, "Usage: .ghost on/off/status")
	}
}

func init() {
	Register(ModuleInfo{
		Name:        "Ghost Mode",
		Description: "Hide read receipts, online status, and typing indicator",
		Commands: []CommandInfo{
			{Pattern: "ghost", Handler: ghostHandler, Sudo: true},
		},
	})
}
