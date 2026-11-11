package modules

import (
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

// Status: set/get the bot's WhatsApp about/bio text.
//
// .setstatus <text>  — set bot's about/status text
// .getstatus         — show current about text
// .clearstatus       — clear the status

func setStatusHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	if len(args) == 0 {
		return ReplyErr(v, "Usage: .setstatus <text>\nExample: .setstatus I am a bot 🤖")
	}

	text := strings.Join(args, " ")
	if len(text) > 139 {
		return ReplyErr(v, fmt.Sprintf("❌ Status too long (%d/139 chars).", len(text)))
	}

	ctx, cancel := newContext(15)
	defer cancel()

	err := wac.SetStatusMessage(ctx, text)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to set status: %v", err))
	}

	return ReplyErr(v, fmt.Sprintf("✅ Status updated:\n_%s_", text))
}

func getStatusHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	if wac.Store.ID == nil {
		return ReplyErr(v, "❌ Bot not logged in.")
	}

	ctx, cancel := newContext(15)
	defer cancel()

	// GetUserInfo fetches profile info including status
	info, err := wac.GetUserInfo(ctx, []string{wac.Store.ID.ToNonAD().String()})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to get status: %v", err))
	}

	botJID := wac.Store.ID.ToNonAD().String()
	if ui, ok := info[botJID]; ok {
		if ui.Status == "" {
			return ReplyErr(v, "ℹ️ No status set.")
		}
		return ReplyErr(v, fmt.Sprintf("📝 *Current status:*\n_%s_", ui.Status))
	}

	return ReplyErr(v, "ℹ️ Could not retrieve status.")
}

func clearBotStatusHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	ctx, cancel := newContext(15)
	defer cancel()

	err := wac.SetStatusMessage(ctx, "")
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to clear status: %v", err))
	}

	return ReplyErr(v, "✅ Status cleared.")
}

func init() {
	Register(ModuleInfo{
		Name:        "Status",
		Description: "Set or get the bot's WhatsApp about/bio text",
		Commands: []CommandInfo{
			{Pattern: "setstatus", Handler: setStatusHandler, Sudo: true},
			{Pattern: "getstatus", Handler: getStatusHandler, Sudo: true},
		},
	})
}
