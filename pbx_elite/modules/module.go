package modules

import (
	"context"
	"log"
	"strings"
	"time"

	"PbxGb/client"
	"PbxGb/config"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// ── CONVENIENCE HELPERS ────────────────────────────────────────────────────

// getClientSafe returns the whatsmeow client or nil if not yet ready.
func getClientSafe() *whatsmeow.Client {
	wac := client.Get()
	if wac == nil {
		return nil
	}
	if !wac.IsConnected() {
		return nil
	}
	return wac
}

// newContext returns a context.WithTimeout and its cancel func.
// n is the timeout in seconds.
func newContext(n int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(n)*time.Second)
}

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
	// ── side-effect hooks (run on every message, no prefix needed) ──
	// — from original PBX Elite modules —
	CheckFlood(v)        // antiflood.go     — kick/warn flood offenders
	TrackMessage(v)      // groupstats.go    — track per-member group stats
	ProcessAIMode(v)     // chatbot.go       — reply when AI mode is on
	MarkAsRead(v)        // autoread.go      — mark read when autoread is on
	TrackStatus(v)       // statushistory.go — record status@broadcast entries
	TrackUserStats(v)    // mystats.go       — per-user message statistics

	// — from newly added modules —
	CheckAFK(v)          // afk.go      — auto-reply when AFK; cancel on own msg
	CheckAntiSpam(v)     // antispam.go — kick/warn spammers
	CheckAntiLink(v)     // antilink.go — kick/warn link posters
	CheckAutoPilot(v)    // autopilot.go — per-chat keyword auto-reply
	ProcessAutoReply(v)  // autoreply.go — global keyword auto-reply (.ar rules)
	CheckMute(v)         // mute.go     — drop messages from muted users
	CheckSlowMode(v)     // slowmode.go — rate-limit messages
	CheckMediaFilter(v)  // mediafilter.go — block disallowed media types
	CheckFilters(v)      // filters.go  — keyword-based auto-responses
	CheckAntibot(v)      // antibot.go  — kick bots that join
	TrackMentions(v)     // mentioned.go — track @mentions for the bot
	CheckReplyRaid(v)   // replyraid.go — auto-reply raid on target users

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

	// Gate: if chat is locked, only owner/sudo can run commands
	if IsChatLocked(v.Info.Chat.String()) && !isSudoOrOwner(v) {
		return
	}

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

// ── GROUP PARTICIPANT EVENTS ───────────────────────────────────────────────

// HandleGroupParticipant is called from main when a group participant event fires.
// Wire it: wac.AddEventHandler(func(evt interface{}) { if e, ok := evt.(*events.GroupInfo); ok { modules.HandleGroupParticipant(e) } })
func HandleGroupParticipant(evt *events.GroupInfo) {
	// welcome/goodbye hooks
	if len(evt.Join) > 0 {
		HandleGroupJoin(evt)  // welcome.go — send welcome message
	}
	if len(evt.Leave) > 0 {
		HandleGroupLeave(evt) // welcome.go — send goodbye message
	}

	// auto-kick banned users who rejoin
	if len(evt.Join) > 0 {
		wac := getClientSafe()
		if wac == nil {
			return
		}
		chat := evt.JID.String()
		for _, jid := range evt.Join {
			if IsBanned(chat, jid.String()) {
				ctx, cancel := newContext(10)
				_, _ = wac.UpdateGroupParticipants(ctx, evt.JID, []types.JID{jid}, "remove")
				cancel()
				log.Printf("[BAN] Auto-kicked rejoining banned user %s from %s", jid.String(), chat)
			}
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
