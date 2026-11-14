package modules

import (
	"context"
	"fmt"
	"strings"
	"time"


	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// .setbio <text>   — set bot's WhatsApp about/bio
// .getbio          — get bot's current bio


func getBioHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if wac.Store.ID == nil {
		return ReplyErr(v, "❌ Bot JID not available.")
	}

	info, err := wac.GetUserInfo(ctx, []types.JID{wac.Store.ID.ToNonAD()})
	if err != nil || len(info) == 0 {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to get bio: %v", err))
	}

	var bio string
	for _, u := range info {
		bio = u.Status
		break
	}

	if bio == "" {
		return ReplyErr(v, "ℹ️ No bio set.")
	}

	return ReplyErr(v, fmt.Sprintf("📝 *Current Bio:*\n_%s_", bio))
}

func init() {
	Register(ModuleInfo{
		Name:        "Bio",
		Description: "Get or set the bot's WhatsApp about/bio",
		Commands: []CommandInfo{
			{Pattern: "getbio", Handler: getBioHandler, Sudo: true},
		},
	})
}
