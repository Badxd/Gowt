package modules

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"PbxGb/client"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	antilinkMu     sync.RWMutex
	antilinkGroups = make(map[string]bool)
	antilinkWarn   = make(map[string]int)
)

// Blocked patterns — http/https, t.me, instagram, wa.me, whatsapp, youtube sab
var linkRegex = regexp.MustCompile(
	`(?i)(https?://` +
		`|t\.me/` +
		`|telegram\.me/` +
		`|wa\.me/` +
		`|chat\.whatsapp\.com/` +
		`|whatsapp\.com/` +
		`|instagram\.com/` +
		`|instagr\.am/` +
		`|youtu\.be/` +
		`|youtube\.com/` +
		`|bit\.ly/` +
		`|tinyurl\.com/)`,
)

func HandleAntiLink(v *events.Message) {
	if !isGroup(v) {
		return
	}
	if isOwner(v) || v.Info.IsFromMe {
		return
	}

	groupStr := v.Info.Chat.String()
	antilinkMu.RLock()
	enabled := antilinkGroups[groupStr]
	antilinkMu.RUnlock()
	if !enabled {
		return
	}

	// Text + caption dono check karo
	text := extractText(v)
	if text == "" && v.Message != nil {
		if img := v.Message.GetImageMessage(); img != nil {
			text = img.GetCaption()
		} else if vid := v.Message.GetVideoMessage(); vid != nil {
			text = vid.GetCaption()
		}
	}
	if !linkRegex.MatchString(text) {
		return
	}

	wac := client.Get()

	// Admins exempt ne
	info, err := wac.GetGroupInfo(context.Background(), v.Info.Chat)
	if err == nil {
		for _, p := range info.Participants {
			if p.JID.User == v.Info.Sender.User && (p.IsAdmin || p.IsSuperAdmin) {
				return
			}
		}
	}

	// Message delete karo
	_, _ = wac.RevokeMessage(context.Background(), v.Info.Chat, v.Info.ID)

	// Warn increment
	warnKey := groupStr + "|" + v.Info.Sender.String()
	antilinkMu.Lock()
	antilinkWarn[warnKey]++
	warns := antilinkWarn[warnKey]
	antilinkMu.Unlock()

	senderJID := v.Info.Sender
	user := senderJID.User
	mentions := []string{senderJID.String()}

	if warns >= 1 {
		// Kick + gban
		antilinkMu.Lock()
		if gbanned[groupStr] == nil {
			gbanned[groupStr] = make(map[string]bool)
		}
		gbanned[groupStr][senderJID.String()] = true
		delete(antilinkWarn, warnKey)
		antilinkMu.Unlock()

		_, _ = wac.UpdateGroupParticipants(
			context.Background(),
			v.Info.Chat,
			[]types.JID{senderJID},
			"remove",
		)

		msg := fmt.Sprintf(
			"🚫 *Anti-Link*\n\n"+
				"👤 @%s\n"+
				"⛔ Sharing links is not allowed in this group!\n"+
				"❌ User kicked & banned.", user)
		sendGroupMention(v.Info.Chat, msg, mentions)
	}
}

// sendGroupMention helper
func sendGroupMention(chatJID types.JID, text string, mentions []string) {
	wac := client.Get()
	_, _ = wac.SendMessage(context.Background(), chatJID, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: &text,
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: mentions,
			},
		},
	})
}

func antilinkToggleHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	groupStr := v.Info.Chat.String()

	if len(args) == 0 {
		antilinkMu.RLock()
		s := antilinkGroups[groupStr]
		antilinkMu.RUnlock()
		state := "🔴 OFF"
		if s {
			state = "🟢 ON"
		}
		return ReplyErr(v, fmt.Sprintf("🔗 Anti-Link: *%s*", state))
	}

	switch strings.ToLower(args[0]) {
	case "on":
		antilinkMu.Lock()
		antilinkGroups[groupStr] = true
		antilinkMu.Unlock()
		return ReplyErr(v,
			"✅ *Anti-Link ON* 🔗\n\n"+
				"Blocked links:\n"+
				"• http / https\n"+
				"• t.me (Telegram)\n"+
				"• instagram.com\n"+
				"• wa.me / chat.whatsapp.com\n"+
				"• youtube.com / youtu.be\n"+
				"• bit.ly / tinyurl\n\n"+
				"Link bhejne pe:\n"+
				"🗑️ Message delete\n"+
				"🚫 User kicked & banned")
	case "off":
		antilinkMu.Lock()
		antilinkGroups[groupStr] = false
		antilinkMu.Unlock()
		return ReplyErr(v, "🔴 *Anti-Link OFF*")
	}
	return ReplyErr(v, "❌ Use: .antilink on / off")
}

func init() {
	Register(ModuleInfo{
		Name:        "AntiLink",
		Description: "Delete links + ban sender in groups",
		Commands: []CommandInfo{
			{
				Pattern: "antilink",
				Handler: antilinkToggleHandler,
				Sudo:    true,
			},
		},
	})
}
