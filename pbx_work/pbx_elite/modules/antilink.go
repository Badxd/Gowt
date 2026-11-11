package modules

import (
	"fmt"
	"strings"
	"sync"

	"PbxGb/database"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// Anti-link: auto-delete WhatsApp group invite links from non-admins.
//
// .antilink on/off/status  — toggle anti-link for this chat
// .allowlink @user         — whitelist a user (they can send links)
// .disallowlink @user      — remove whitelist

var (
	antilinkMu      sync.RWMutex
	antilinkEnabled = make(map[string]bool)     // chatJID -> enabled
	antilinkAllow   = make(map[string][]string) // chatJID -> []allowedUserJID (normalized)
)

// linkPatterns are patterns that match WhatsApp invite links.
var linkPatterns = []string{
	"chat.whatsapp.com/",
	"wa.me/",
	"whatsapp.com/",
}

func isGroupLink(text string) bool {
	lower := strings.ToLower(text)
	for _, pat := range linkPatterns {
		if strings.Contains(lower, pat) {
			return true
		}
	}
	return false
}

func isAllowedSender(chat, sender string) bool {
	norm := normalizeJIDForSudo(sender)
	antilinkMu.RLock()
	list := antilinkAllow[chat]
	antilinkMu.RUnlock()
	for _, a := range list {
		if a == norm {
			return true
		}
	}
	return false
}

// CheckAntiLink is the exported hook — auto-deletes WhatsApp links from non-admins.
func CheckAntiLink(v *events.Message) {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return
	}

	chat := v.Info.Chat.String()
	antilinkMu.RLock()
	enabled := antilinkEnabled[chat]
	antilinkMu.RUnlock()
	if !enabled {
		return
	}

	wac := getClientSafe()
	if wac == nil {
		return
	}

	// skip own messages
	if wac.Store.ID != nil && strings.HasPrefix(v.Info.Sender.String(), wac.Store.ID.User) {
		return
	}

	body := extractBodyText(v)
	if !isGroupLink(body) {
		return
	}

	// allow if sender is a group admin — fetch group info
	ctx, cancel := newContext(10)
	groupInfo, err := wac.GetGroupInfo(v.Info.Chat)
	cancel()
	if err == nil {
		for _, p := range groupInfo.Participants {
			if p.JID.User == v.Info.Sender.User && (p.IsAdmin || p.IsSuperAdmin) {
				return // sender is admin, skip
			}
		}
	}

	// allow if sender is in the allowlist
	if isAllowedSender(chat, v.Info.Sender.String()) {
		return
	}

	// revoke the message
	ctx2, cancel2 := newContext(8)
	_ = wac.RevokeMessage(ctx2, v.Info.Chat, false, v.Info.ID, "", "")
	cancel2()

	// warn the user
	phone := v.Info.Sender.User
	mentions := []string{v.Info.Sender.String()}
	warn := fmt.Sprintf("🚫 @%s, group invite links are not allowed here. Your message has been removed.", phone)
	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(warn),
			ContextInfo: &waProto.ContextInfo{
				MentionedJid: mentions,
			},
		},
	}
	ctx3, cancel3 := newContext(8)
	defer cancel3()
	_, _ = wac.SendMessage(ctx3, v.Info.Chat, msg)
}

func antilinkHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	chat := v.Info.Chat.String()

	if len(args) == 0 {
		antilinkMu.RLock()
		en := antilinkEnabled[chat]
		antilinkMu.RUnlock()
		status := "🔴 disabled"
		if en {
			status = "🟢 enabled"
		}
		return ReplyErr(v, fmt.Sprintf("🔗 Anti-link is currently *%s* in this chat.\nUse .antilink on/off to toggle.", status))
	}

	switch strings.ToLower(args[0]) {
	case "on", "enable":
		antilinkMu.Lock()
		antilinkEnabled[chat] = true
		antilinkMu.Unlock()
		_ = database.SetToggle("antilink", chat, true)
		return ReplyErr(v, "✅ Anti-link *enabled*. Group invite links from non-admins will be auto-deleted.")
	case "off", "disable":
		antilinkMu.Lock()
		antilinkEnabled[chat] = false
		antilinkMu.Unlock()
		_ = database.SetToggle("antilink", chat, false)
		return ReplyErr(v, "✅ Anti-link *disabled*.")
	case "status":
		antilinkMu.RLock()
		en := antilinkEnabled[chat]
		antilinkMu.RUnlock()
		status := "🔴 disabled"
		if en {
			status = "🟢 enabled"
		}
		return ReplyErr(v, fmt.Sprintf("🔗 Anti-link: *%s*", status))
	default:
		return ReplyErr(v, "Usage: .antilink on/off/status")
	}
}

func allowLinkHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .allowlink @mention")
	}

	chat := v.Info.Chat.String()
	norm := normalizeJIDForSudo(target)

	antilinkMu.Lock()
	list := antilinkAllow[chat]
	for _, a := range list {
		if a == norm {
			antilinkMu.Unlock()
			return ReplyErr(v, "ℹ️ User already in the allowlist.")
		}
	}
	antilinkAllow[chat] = append(list, norm)
	antilinkMu.Unlock()

	phone := strings.Split(norm, "@")[0]
	return ReplyErr(v, fmt.Sprintf("✅ *+%s* can now send links in this group.", phone))
}

func disallowLinkHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .disallowlink @mention")
	}

	chat := v.Info.Chat.String()
	norm := normalizeJIDForSudo(target)

	antilinkMu.Lock()
	list := antilinkAllow[chat]
	newList := list[:0]
	found := false
	for _, a := range list {
		if a == norm {
			found = true
			continue
		}
		newList = append(newList, a)
	}
	antilinkAllow[chat] = newList
	antilinkMu.Unlock()

	phone := strings.Split(norm, "@")[0]
	if !found {
		return ReplyErr(v, fmt.Sprintf("ℹ️ *+%s* was not in the allowlist.", phone))
	}
	return ReplyErr(v, fmt.Sprintf("✅ *+%s* removed from link allowlist.", phone))
}

func init() {
	// Restore persisted antilink toggles from MongoDB.
	loaded := database.LoadAllToggles("antilink")
	antilinkMu.Lock()
	for chatJID, enabled := range loaded {
		antilinkEnabled[chatJID] = enabled
	}
	antilinkMu.Unlock()

	Register(ModuleInfo{
		Name:        "Anti-Link",
		Description: "Auto-delete WhatsApp group invite links from non-admins",
		Commands: []CommandInfo{
			{Pattern: "antilink", Handler: antilinkHandler, Sudo: true},
			{Pattern: "allowlink", Handler: allowLinkHandler, Sudo: true},
			{Pattern: "disallowlink", Handler: disallowLinkHandler, Sudo: true},
		},
	})
}
