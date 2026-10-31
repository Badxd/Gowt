package modules

import (
	"log"
	"strings"
	"time"

	"PbxGb/client"
	"PbxGb/config"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type CommandInfo struct {
	Pattern string
	Handler func(*events.Message, []string) error
	Sudo    bool
}

type ModuleInfo struct {
	Name        string
	Description string
	Commands    []CommandInfo
}

var allModules []ModuleInfo

func Register(m ModuleInfo) {
	allModules = append(allModules, m)
}

// ── OWNER CHECK ────────────────────────────────────────────────────────────

func isOwner(v *events.Message) bool {

	// PRIMARY: IsFromMe flag (most reliable)
	if v.Info.IsFromMe {
		log.Printf("[AUTH] ✅ Owner via IsFromMe=true")
		return true
	}

	wac := client.Get()

	// SECONDARY: Bot's own JID
	if wac.Store.ID != nil && wac.Store.ID.User == v.Info.Sender.User {
		log.Printf("[AUTH] ✅ Owner via Store.ID match: %s", v.Info.Sender.User)
		return true
	}

	// TERTIARY: LID match
	if wac.Store.LID.User != "" && wac.Store.LID.User == v.Info.Sender.User {
		log.Printf("[AUTH] ✅ Owner via Store.LID match: %s", v.Info.Sender.User)
		return true
	}

	// QUATERNARY: Phone number fallback
	ownerPhone := ownerNum()
	if v.Info.Sender.User == ownerPhone {
		log.Printf("[AUTH] ✅ Owner via phone match: %s", v.Info.Sender.User)
		return true
	}

	return false
}

// ownerNum — pure phone number from config
func ownerNum() string {
	n := config.C.OwnerJID
	if idx := strings.Index(n, "@"); idx != -1 {
		return n[:idx]
	}
	return n
}

// isSudoOrOwner
func isSudoOrOwner(v *events.Message) bool {
	if isOwner(v) {
		return true
	}
	return isSudoSender(v)
}

// ── HANDLE MESSAGE ─────────────────────────────────────────────────────────

func HandleMessage(v *events.Message) {

	text := extractText(v)
	if text == "" {
		return
	}

	prefix := config.C.Prefix
	if !strings.HasPrefix(text, prefix) {
		return
	}

	parts := strings.Fields(strings.TrimPrefix(text, prefix))
	if len(parts) == 0 {
		return
	}

	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	for _, mod := range allModules {
		for _, command := range mod.Commands {
			if !strings.EqualFold(command.Pattern, cmd) {
				continue
			}

			// Only owner/sudo-restricted commands actually check that;
			// Sudo:false commands are open to anyone who can message here.
			if command.Sudo {
				ownerMatch := isOwner(v)
				sudoMatch := !ownerMatch && isSudoSender(v)
				if !ownerMatch && !sudoMatch {
					return
				}
			}

			_ = command.Handler(v, args)
			return
		}
	}
}

// ── HELPERS ────────────────────────────────────────────────────────────────

func safeSendMessage(v *events.Message, text string) {
	_, err := Reply(v, text)
	if err != nil {
		time.Sleep(5 * time.Second)
		_, _ = Reply(v, text)
	}
}

func extractText(v *events.Message) string {
	msg := v.Message
	if msg == nil {
		return ""
	}
	if msg.Conversation != nil {
		return *msg.Conversation
	}
	if msg.ExtendedTextMessage != nil {
		return msg.ExtendedTextMessage.GetText()
	}
	return ""
}

func IsDirectMessage(v *events.Message) bool {
	return v.Info.Chat.Server == types.DefaultUserServer ||
		v.Info.Chat.Server == types.HiddenUserServer
}
