package modules

import (
	"fmt"
	"strings"
	"sync"
	"time"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// Anti-spam: detect repeated messages or flood bursts and mute/kick offenders.
//
// .antispam on/off/status         — toggle anti-spam for this chat
// .antispam limit <n>             — max same messages allowed (default 3)
// .antispam interval <seconds>    — time window in seconds (default 10)
// .antispam action mute/kick      — action to take (default: mute for 60s)

type spamRecord struct {
	LastMsg   string
	Count     int
	WindowEnd time.Time
}

type antispamConfig struct {
	Enabled  bool
	Limit    int // max identical messages in window
	Interval int // seconds window
	Action   string
}

var (
	antispamMu      sync.RWMutex
	antispamConfigs = make(map[string]*antispamConfig) // chatJID -> config
	spamTracker     = make(map[string]*spamRecord)     // "chatJID|userJID" -> record
)

func getAntispamConfig(chat string) *antispamConfig {
	antispamMu.RLock()
	cfg, ok := antispamConfigs[chat]
	antispamMu.RUnlock()
	if !ok || cfg == nil {
		return &antispamConfig{Limit: 3, Interval: 10, Action: "mute"}
	}
	return cfg
}

// CheckAntiSpam is the exported hook — detects and handles message spam.
func CheckAntiSpam(v *events.Message) {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return
	}

	chat := v.Info.Chat.String()
	cfg := getAntispamConfig(chat)
	if !cfg.Enabled {
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

	// skip group admins
	ctx, cancel := newContext(8)
	groupInfo, err := wac.GetGroupInfo(ctx, v.Info.Chat)
	cancel()
	if err == nil {
		for _, p := range groupInfo.Participants {
			if p.JID.User == v.Info.Sender.User && (p.IsAdmin || p.IsSuperAdmin) {
				return
			}
		}
	}

	body := strings.TrimSpace(strings.ToLower(extractBodyText(v)))
	if body == "" {
		return
	}

	key := chat + "|" + v.Info.Sender.String()
	now := time.Now()

	antispamMu.Lock()
	rec, ok := spamTracker[key]
	if !ok || now.After(rec.WindowEnd) {
		rec = &spamRecord{
			LastMsg:   body,
			Count:     1,
			WindowEnd: now.Add(time.Duration(cfg.Interval) * time.Second),
		}
		spamTracker[key] = rec
		antispamMu.Unlock()
		return
	}

	if body == rec.LastMsg {
		rec.Count++
	} else {
		rec.LastMsg = body
		rec.Count = 1
		rec.WindowEnd = now.Add(time.Duration(cfg.Interval) * time.Second)
	}

	count := rec.Count
	limit := cfg.Limit
	action := cfg.Action
	antispamMu.Unlock()

	if count < limit {
		return
	}

	// reset tracker
	antispamMu.Lock()
	delete(spamTracker, key)
	antispamMu.Unlock()

	phone := v.Info.Sender.User
	mentions := []string{v.Info.Sender.String()}

	switch action {
	case "kick":
		targetJID, parseErr := types.ParseJID(v.Info.Sender.String())
		if parseErr == nil {
			ctx2, cancel2 := newContext(10)
			_, _ = wac.UpdateGroupParticipants(ctx2, v.Info.Chat, []types.JID{targetJID}, "remove")
			cancel2()
		}
		ctx3, cancel3 := newContext(10)
		defer cancel3()
		warn := fmt.Sprintf("🚫 @%s was kicked for spamming.", phone)
		msg := &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text:        proto.String(warn),
				ContextInfo: &waProto.ContextInfo{MentionedJID: mentions},
			},
		}
		_, _ = wac.SendMessage(ctx3, v.Info.Chat, msg)

	default: // "mute" — warn and delete their message
		ctx2, cancel2 := newContext(8)
		_, _ = wac.RevokeMessage(ctx2, v.Info.Chat, v.Info.ID)
		cancel2()

		ctx3, cancel3 := newContext(8)
		defer cancel3()
		warn := fmt.Sprintf("⚠️ @%s, please stop spamming! (%d/%d identical messages in %ds window). Message removed.",
			phone, count, limit, cfg.Interval)
		msg := &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text:        proto.String(warn),
				ContextInfo: &waProto.ContextInfo{MentionedJID: mentions},
			},
		}
		_, _ = wac.SendMessage(ctx3, v.Info.Chat, msg)
	}
}

func antispamHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	chat := v.Info.Chat.String()

	if len(args) == 0 {
		cfg := getAntispamConfig(chat)
		status := "🔴 disabled"
		if cfg.Enabled {
			status = "🟢 enabled"
		}
		return ReplyErr(v, fmt.Sprintf(
			"🛡️ Anti-spam: *%s*\n📊 Limit: *%d* same msgs\n⏱️ Window: *%ds*\n⚡ Action: *%s*\n\nSubcommands: on/off/status, limit <n>, interval <s>, action mute/kick",
			status, cfg.Limit, cfg.Interval, cfg.Action))
	}

	antispamMu.Lock()
	if antispamConfigs[chat] == nil {
		antispamConfigs[chat] = &antispamConfig{Limit: 3, Interval: 10, Action: "mute"}
	}
	cfg := antispamConfigs[chat]
	antispamMu.Unlock()

	sub := strings.ToLower(args[0])

	switch sub {
	case "on", "enable":
		antispamMu.Lock()
		cfg.Enabled = true
		antispamMu.Unlock()
		return ReplyErr(v, "✅ Anti-spam *enabled*.")
	case "off", "disable":
		antispamMu.Lock()
		cfg.Enabled = false
		antispamMu.Unlock()
		return ReplyErr(v, "✅ Anti-spam *disabled*.")
	case "status":
		antispamMu.RLock()
		en := cfg.Enabled
		antispamMu.RUnlock()
		status := "🔴 disabled"
		if en {
			status = "🟢 enabled"
		}
		return ReplyErr(v, fmt.Sprintf("🛡️ Anti-spam: *%s*", status))
	case "limit":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .antispam limit <n>  (e.g. .antispam limit 5)")
		}
		var n int
		if _, err := fmt.Sscan(args[1], &n); err != nil || n < 2 || n > 20 {
			return ReplyErr(v, "❌ Limit must be between 2 and 20.")
		}
		antispamMu.Lock()
		cfg.Limit = n
		antispamMu.Unlock()
		return ReplyErr(v, fmt.Sprintf("✅ Spam limit set to *%d* identical messages.", n))
	case "interval":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .antispam interval <seconds>  (e.g. .antispam interval 10)")
		}
		var n int
		if _, err := fmt.Sscan(args[1], &n); err != nil || n < 3 || n > 300 {
			return ReplyErr(v, "❌ Interval must be between 3 and 300 seconds.")
		}
		antispamMu.Lock()
		cfg.Interval = n
		antispamMu.Unlock()
		return ReplyErr(v, fmt.Sprintf("✅ Spam window set to *%d seconds*.", n))
	case "action":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .antispam action mute/kick")
		}
		act := strings.ToLower(args[1])
		if act != "mute" && act != "kick" {
			return ReplyErr(v, "❌ Action must be 'mute' or 'kick'.")
		}
		antispamMu.Lock()
		cfg.Action = act
		antispamMu.Unlock()
		return ReplyErr(v, fmt.Sprintf("✅ Spam action set to *%s*.", act))
	default:
		return ReplyErr(v, "Usage: .antispam on/off/status/limit/interval/action")
	}
}

func init() {
	Register(ModuleInfo{
		Name:        "Anti-Spam",
		Description: "Detect and handle message spam in groups",
		Commands: []CommandInfo{
			{Pattern: "antispam", Handler: antispamHandler, Sudo: true},
		},
	})
}
