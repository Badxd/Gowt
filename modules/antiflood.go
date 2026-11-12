package modules

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"PbxGb/client"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

// antiFloodConfig holds settings per chat
type floodConfig struct {
	MaxMsgs  int           // max messages allowed
	Window   time.Duration // within this time window
	Action   string        // "warn", "kick", "mute"
	Enabled  bool
}

// floodTracker tracks message timestamps per sender per chat
type floodTracker struct {
	mu      sync.Mutex
	counts  map[string]map[string][]time.Time // chatJID → senderJID → timestamps
	configs map[string]*floodConfig            // chatJID → config
}

var flood = &floodTracker{
	counts:  make(map[string]map[string][]time.Time),
	configs: make(map[string]*floodConfig),
}

func (f *floodTracker) setConfig(chatJID string, cfg *floodConfig) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.configs[chatJID] = cfg
}

func (f *floodTracker) getConfig(chatJID string) *floodConfig {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.configs[chatJID]
}

func (f *floodTracker) delConfig(chatJID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.configs, chatJID)
}

// Check returns true if the sender is flooding
func (f *floodTracker) Check(chatJID, senderJID string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	cfg, ok := f.configs[chatJID]
	if !ok || !cfg.Enabled {
		return false
	}

	now := time.Now()
	if f.counts[chatJID] == nil {
		f.counts[chatJID] = make(map[string][]time.Time)
	}

	// Filter old timestamps
	var fresh []time.Time
	for _, t := range f.counts[chatJID][senderJID] {
		if now.Sub(t) <= cfg.Window {
			fresh = append(fresh, t)
		}
	}
	fresh = append(fresh, now)
	f.counts[chatJID][senderJID] = fresh

	return len(fresh) > cfg.MaxMsgs
}

// CheckMessage is called from the global message hook
func CheckFlood(v *events.Message) {
	chatJID := v.Info.Chat.String()
	senderJID := SenderJID(v)

	if !flood.Check(chatJID, senderJID) {
		return
	}

	cfg := flood.getConfig(chatJID)
	if cfg == nil {
		return
	}

	wac := client.Get()
	warn := fmt.Sprintf("⚠️ @%s slow down! You're flooding the chat.", senderJID)

	switch cfg.Action {
	case "warn":
		wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: &warn,
				ContextInfo: &waProto.ContextInfo{
					MentionedJID: []string{senderJID},
				},
			},
		})
	case "delete":
		_, _ = wac.RevokeMessage(context.Background(), v.Info.Chat, v.Info.ID)
	}
}

// .antiflood <max> <seconds> [warn|delete]
func antiFloodHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		return ReplyErr(v, "Usage: .antiflood <max_msgs> <seconds> [warn|delete]\nExample: .antiflood 5 10 warn")
	}
	max, err := strconv.Atoi(args[0])
	if err != nil || max <= 0 {
		return ReplyErr(v, "❌ Invalid max_msgs.")
	}
	secs, err := strconv.Atoi(args[1])
	if err != nil || secs <= 0 {
		return ReplyErr(v, "❌ Invalid seconds.")
	}
	action := "warn"
	if len(args) >= 3 && (args[2] == "delete" || args[2] == "warn") {
		action = args[2]
	}

	jid := v.Info.Chat.String()
	flood.setConfig(jid, &floodConfig{
		MaxMsgs: max,
		Window:  time.Duration(secs) * time.Second,
		Action:  action,
		Enabled: true,
	})

	return ReplyErr(v, fmt.Sprintf("✅ Anti-flood enabled: max *%d* msgs per *%ds*, action: *%s*", max, secs, action))
}

// .stopantiflood
func stopAntiFloodHandler(v *events.Message, args []string) error {
	jid := v.Info.Chat.String()
	flood.delConfig(jid)
	return ReplyErr(v, "✅ Anti-flood disabled for this chat.")
}

func init() {
	Register(ModuleInfo{
		Name:        "AntiFlood",
		Description: "Detect and act on message flooding in groups",
		Commands: []CommandInfo{
			{Pattern: "antiflood", Handler: antiFloodHandler, Sudo: true},
			{Pattern: "stopantiflood", Handler: stopAntiFloodHandler, Sudo: true},
		},
	})
}
