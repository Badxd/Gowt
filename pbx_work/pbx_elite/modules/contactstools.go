package modules

// Contacts Tools — list contacts, block/unblock numbers.
//
// Commands:
//   .contacts              — list all saved contacts (count + first 20)
//   .searchcontact <name>  — search contacts by name
//   .block @user|number    — block a WhatsApp number
//   .unblock @user|number  — unblock a number
//   .blocklist             — show your block list
//   .isblocked @user       — check if a number is blocked
//   .contactinfo @user     — show full contact info from local store

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"PbxGb/client"
)

// ── LIST CONTACTS ─────────────────────────────────────────────────────────────
func listContactsHandler(v *events.Message, args []string) error {
	wac := client.Get()
	all, err := wac.Store.Contacts.GetAllContacts()
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	var contacts []string
	for jid, info := range all {
		if jid.Server != types.DefaultUserServer {
			continue
		}
		name := info.FullName
		if name == "" {
			name = info.PushName
		}
		if name == "" {
			name = "+"+jid.User
		}
		contacts = append(contacts, fmt.Sprintf("👤 *%s* — +%s", name, jid.User))
		if len(contacts) >= 20 {
			break
		}
	}

	total := 0
	for jid := range all {
		if jid.Server == types.DefaultUserServer {
			total++
		}
	}

	if len(contacts) == 0 {
		return ReplyErr(v, "ℹ️ No contacts found in store.")
	}

	header := fmt.Sprintf("📒 *Contacts (%d total — showing first %d):*\n\n", total, len(contacts))
	return ReplyErr(v, header+strings.Join(contacts, "\n"))
}

// ── SEARCH CONTACTS ───────────────────────────────────────────────────────────
func searchContactHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .searchcontact <name or number>")
	}
	query := strings.ToLower(strings.Join(args, " "))
	wac := client.Get()
	all, err := wac.Store.Contacts.GetAllContacts()
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	var results []string
	for jid, info := range all {
		if jid.Server != types.DefaultUserServer {
			continue
		}
		name := strings.ToLower(info.FullName + " " + info.PushName)
		num := jid.User
		if strings.Contains(name, query) || strings.Contains(num, query) {
			displayName := info.FullName
			if displayName == "" {
				displayName = info.PushName
			}
			if displayName == "" {
				displayName = "+"+num
			}
			results = append(results, fmt.Sprintf("👤 *%s* — +%s", displayName, num))
		}
		if len(results) >= 15 {
			break
		}
	}

	if len(results) == 0 {
		return ReplyErr(v, fmt.Sprintf("🔍 No contacts matched *%s*.", query))
	}
	return ReplyErr(v, fmt.Sprintf("🔍 *Search results for \"%s\":*\n\n%s", query, strings.Join(results, "\n")))
}

// ── BLOCK ─────────────────────────────────────────────────────────────────────
func blockHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .block @mention or .block 919xxxxxxxx")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}

	wac := client.Get()
	ctx := context.Background()

	resp, err := wac.UpdateBlocklist(ctx, jid, "block")
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Block failed: %v", err))
	}
	_ = resp
	return ReplyErr(v, fmt.Sprintf("🚫 Blocked *+%s* successfully.", jid.User))
}

// ── UNBLOCK ───────────────────────────────────────────────────────────────────
func unblockHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .unblock @mention or .unblock 919xxxxxxxx")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}

	wac := client.Get()
	ctx := context.Background()

	resp, err := wac.UpdateBlocklist(ctx, jid, "unblock")
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Unblock failed: %v", err))
	}
	_ = resp
	return ReplyErr(v, fmt.Sprintf("✅ Unblocked *+%s* successfully.", jid.User))
}

// ── BLOCK LIST ────────────────────────────────────────────────────────────────
func blockListHandler(v *events.Message, args []string) error {
	wac := client.Get()
	ctx := context.Background()

	resp, err := wac.GetBlocklist(ctx)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to get blocklist: %v", err))
	}

	if len(resp.JIDs) == 0 {
		return ReplyErr(v, "ℹ️ Your block list is empty.")
	}

	var lines []string
	for i, jid := range resp.JIDs {
		name := ""
		if contact, cerr := wac.Store.Contacts.GetContact(jid); cerr == nil {
			name = contact.FullName
			if name == "" {
				name = contact.PushName
			}
		}
		if name == "" {
			name = "+"+jid.User
		}
		lines = append(lines, fmt.Sprintf("%d. 🚫 *%s* (+%s)", i+1, name, jid.User))
	}

	return ReplyErr(v, fmt.Sprintf("🚫 *Block List (%d):*\n\n%s", len(resp.JIDs), strings.Join(lines, "\n")))
}

// ── IS BLOCKED ────────────────────────────────────────────────────────────────
func isBlockedHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .isblocked @mention or .isblocked 919xxx")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}

	wac := client.Get()
	ctx := context.Background()

	resp, err := wac.GetBlocklist(ctx)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	for _, blocked := range resp.JIDs {
		if blocked.User == jid.User {
			return ReplyErr(v, fmt.Sprintf("🚫 *+%s* is in your block list.", jid.User))
		}
	}
	return ReplyErr(v, fmt.Sprintf("✅ *+%s* is NOT blocked.", jid.User))
}

// ── CONTACT INFO ──────────────────────────────────────────────────────────────
func contactInfoHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .contactinfo @mention or .contactinfo 919xxx")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}

	wac := client.Get()
	ctx := context.Background()

	// Local store info
	contact, storeErr := wac.Store.Contacts.GetContact(jid)

	// Live WA info
	userInfoMap, uiErr := wac.GetUserInfo(ctx, []types.JID{jid})

	lines := []string{fmt.Sprintf("📋 *Contact Info — +%s*\n", jid.User)}

	if storeErr == nil {
		if contact.FullName != "" {
			lines = append(lines, fmt.Sprintf("👤 *Full Name:* %s", contact.FullName))
		}
		if contact.FirstName != "" && contact.FirstName != contact.FullName {
			lines = append(lines, fmt.Sprintf("🔤 *First Name:* %s", contact.FirstName))
		}
		if contact.PushName != "" {
			lines = append(lines, fmt.Sprintf("📲 *Push Name:* %s", contact.PushName))
		}
		if contact.BusinessName != "" {
			lines = append(lines, fmt.Sprintf("🏢 *Business Name:* %s", contact.BusinessName))
		}
	}

	if uiErr == nil {
		if info, ok := userInfoMap[jid]; ok {
			if info.VerifiedName != nil {
				lines = append(lines, fmt.Sprintf("✅ *Verified Business:* %s", info.VerifiedName.Name))
			}
			if info.IsBusiness {
				lines = append(lines, "💼 *Account Type:* WhatsApp Business")
			} else {
				lines = append(lines, "👤 *Account Type:* Personal")
			}
			status := "⚠️ Unknown"
			switch info.Status {
			case types.UserStatusActive:
				status = "✅ Active"
			case types.UserStatusUnavailable:
				status = "❌ Unavailable / Deleted"
			}
			lines = append(lines, fmt.Sprintf("📶 *Status:* %s", status))
		}
	}

	if len(lines) == 1 {
		lines = append(lines, "_(no info available)_")
	}
	return ReplyErr(v, strings.Join(lines, "\n"))
}

func init() {
	Register(ModuleInfo{
		Name:        "ContactsTools",
		Description: "List/search contacts, block/unblock numbers, view contact info",
		Commands: []CommandInfo{
			{Pattern: "contacts",      Handler: listContactsHandler,  Sudo: true},
			{Pattern: "listcontacts",  Handler: listContactsHandler,  Sudo: true},
			{Pattern: "searchcontact", Handler: searchContactHandler, Sudo: true},
			{Pattern: "findcontact",   Handler: searchContactHandler, Sudo: true},
			{Pattern: "block",         Handler: blockHandler,         Sudo: true},
			{Pattern: "unblock",       Handler: unblockHandler,       Sudo: true},
			{Pattern: "blocklist",     Handler: blockListHandler,     Sudo: true},
			{Pattern: "blocked",       Handler: blockListHandler,     Sudo: true},
			{Pattern: "isblocked",     Handler: isBlockedHandler,     Sudo: true},
			{Pattern: "contactinfo",   Handler: contactInfoHandler,   Sudo: true},
			{Pattern: "cinfo",         Handler: contactInfoHandler,   Sudo: true},
		},
	})
}
