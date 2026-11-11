package modules

// Business Tools — WhatsApp Business profile commands.
//
// Commands:
//   .bizprofile           — show business profile of a contact or yourself
//   .getbizinfo @user     — get business info (category, description, address, email, website)
//   .checkbiz @user       — check if a number is a WhatsApp Business account

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"PbxGb/client"
)

// ── BUSINESS PROFILE ─────────────────────────────────────────────────────────
func bizProfileHandler(v *events.Message, args []string) error {
	wac := client.Get()
	ctx := context.Background()

	var jid types.JID
	var err error

	if len(args) > 0 {
		target, terr := extractTargetJID(v, args)
		if terr != nil || target == "" {
			return ReplyErr(v, "Usage: .bizprofile @mention or .bizprofile 919xxx")
		}
		jid, err = types.ParseJID(target)
		if err != nil {
			return ReplyErr(v, "❌ Invalid JID.")
		}
	} else {
		if wac.Store.ID == nil {
			return ReplyErr(v, "❌ Not connected.")
		}
		jid = *wac.Store.ID
	}

	contacts, err := wac.GetUserInfo(ctx, []types.JID{jid})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	info, ok := contacts[jid]
	if !ok {
		return ReplyErr(v, fmt.Sprintf("ℹ️ No info for *+%s*.", jid.User))
	}

	isBiz := info.VerifiedName != nil || info.IsBusiness

	lines := []string{fmt.Sprintf("💼 *Business Profile — +%s*\n", jid.User)}

	if isBiz {
		lines = append(lines, "✅ WhatsApp Business Account")
	} else {
		lines = append(lines, "👤 Regular WhatsApp Account")
	}

	if info.VerifiedName != nil {
		lines = append(lines, fmt.Sprintf("\n🏢 *Business Name:* %s", info.VerifiedName.Name))
		if info.VerifiedName.Details != nil {
			if info.VerifiedName.Details.GetVerifiedName() != "" {
				lines = append(lines, fmt.Sprintf("✅ *Verified:* %s", info.VerifiedName.Details.GetVerifiedName()))
			}
		}
	}

	// Store contact info
	contact, storeErr := wac.Store.Contacts.GetContact(jid)
	if storeErr == nil {
		if contact.FullName != "" {
			lines = append(lines, fmt.Sprintf("\n📛 *Display Name:* %s", contact.FullName))
		}
		if contact.BusinessName != "" && contact.BusinessName != contact.FullName {
			lines = append(lines, fmt.Sprintf("🏷️ *Saved As:* %s", contact.BusinessName))
		}
	}

	return ReplyErr(v, strings.Join(lines, "\n"))
}

// ── CHECK BUSINESS ────────────────────────────────────────────────────────────
func checkBizHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .checkbiz @mention or .checkbiz 919xxx")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}

	wac := client.Get()
	ctx := context.Background()

	contacts, err := wac.GetUserInfo(ctx, []types.JID{jid})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	info, ok := contacts[jid]
	if !ok {
		return ReplyErr(v, fmt.Sprintf("ℹ️ No info found for *+%s*.", jid.User))
	}

	if info.VerifiedName != nil || info.IsBusiness {
		name := "unknown"
		if info.VerifiedName != nil {
			name = info.VerifiedName.Name
		}
		return ReplyErr(v, fmt.Sprintf("✅ *+%s* is a *WhatsApp Business* account.\n🏢 Name: *%s*", jid.User, name))
	}
	return ReplyErr(v, fmt.Sprintf("👤 *+%s* is a regular WhatsApp account (not Business).", jid.User))
}

func init() {
	Register(ModuleInfo{
		Name:        "BusinessTools",
		Description: "View WhatsApp Business profiles and verify business accounts",
		Commands: []CommandInfo{
			{Pattern: "bizprofile",  Handler: bizProfileHandler, Sudo: true},
			{Pattern: "getbizinfo",  Handler: bizProfileHandler, Sudo: true},
			{Pattern: "checkbiz",    Handler: checkBizHandler,   Sudo: true},
		},
	})
}
