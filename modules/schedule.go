package modules

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// Schedule a message to be sent after a delay.
//
// .schedule <delay> <message>   — schedule a message (delay: 10s, 5m, 1h)
// .schedulelist                 — list pending scheduled messages
// .cancelschedule <id>          — cancel a scheduled message

type scheduleEntry struct {
	ID      int
	ChatJID types.JID
	Text    string
	SendAt  time.Time
	Timer   *time.Timer
}

var (
	scheduleMu      sync.Mutex
	scheduleMap     = make(map[int]*scheduleEntry)
	scheduleCounter = 0
)

func parseDuration(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	// support plain numbers as seconds
	if n, err := strconv.Atoi(s); err == nil {
		return time.Duration(n) * time.Second, nil
	}
	return time.ParseDuration(s)
}

func scheduleHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		return ReplyErr(v, "Usage: .schedule <delay> <message>\nExamples:\n  .schedule 30s Hello everyone!\n  .schedule 5m Meeting in 5 minutes\n  .schedule 1h Time is up!")
	}

	dur, err := parseDuration(args[0])
	if err != nil || dur <= 0 {
		return ReplyErr(v, "❌ Invalid delay. Use formats like: 30s, 5m, 1h, 90s")
	}
	if dur > 24*time.Hour {
		return ReplyErr(v, "❌ Maximum schedule delay is 24 hours.")
	}

	text := strings.Join(args[1:], " ")
	chat := v.Info.Chat

	scheduleMu.Lock()
	scheduleCounter++
	id := scheduleCounter
	entry := &scheduleEntry{
		ID:      id,
		ChatJID: chat,
		Text:    text,
		SendAt:  time.Now().Add(dur),
	}

	entry.Timer = time.AfterFunc(dur, func() {
		wac := getClientSafe()
		if wac == nil {
			return
		}
		msg := &waProto.Message{Conversation: proto.String(text)}
		ctx, cancel := newContext(10)
		defer cancel()
		_, _ = wac.SendMessage(ctx, chat, msg)

		scheduleMu.Lock()
		delete(scheduleMap, id)
		scheduleMu.Unlock()
	})

	scheduleMap[id] = entry
	scheduleMu.Unlock()

	sendAt := time.Now().Add(dur).Format("15:04:05")
	return ReplyErr(v, fmt.Sprintf("⏰ Message scheduled!\n📌 ID: *#%d*\n⏱️ Delay: *%s*\n🕐 Send at: *%s*\n📝 Message: %s", id, dur, sendAt, text))
}

func scheduleListHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()

	scheduleMu.Lock()
	defer scheduleMu.Unlock()

	var lines []string
	lines = append(lines, "⏰ *Scheduled Messages:*", "")

	count := 0
	for _, e := range scheduleMap {
		if e.ChatJID.String() != chat {
			continue
		}
		remaining := time.Until(e.SendAt).Round(time.Second)
		preview := e.Text
		if len(preview) > 40 {
			preview = preview[:40] + "..."
		}
		lines = append(lines, fmt.Sprintf("*#%d* — in %s\n  %s", e.ID, remaining, preview))
		count++
	}

	if count == 0 {
		return ReplyErr(v, "ℹ️ No scheduled messages for this chat.")
	}
	return ReplyErr(v, strings.Join(lines, "\n"))
}

func cancelScheduleHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .cancelschedule <id>\nExample: .cancelschedule 3")
	}

	var id int
	s := strings.TrimPrefix(strings.TrimSpace(args[0]), "#")
	if _, err := fmt.Sscan(s, &id); err != nil || id <= 0 {
		return ReplyErr(v, "❌ Invalid ID. Use the ID from .schedulelist.")
	}

	scheduleMu.Lock()
	entry, ok := scheduleMap[id]
	if ok {
		entry.Timer.Stop()
		delete(scheduleMap, id)
	}
	scheduleMu.Unlock()

	if !ok {
		return ReplyErr(v, fmt.Sprintf("❌ No scheduled message with ID #%d.", id))
	}
	return ReplyErr(v, fmt.Sprintf("✅ Scheduled message *#%d* cancelled.", id))
}

func init() {
	Register(ModuleInfo{
		Name:        "Scheduler",
		Description: "Schedule a message to be sent after a delay",
		Commands: []CommandInfo{
			{Pattern: "schedule", Handler: scheduleHandler, Sudo: true},
			{Pattern: "schedulelist", Handler: scheduleListHandler, Sudo: true},
			{Pattern: "cancelschedule", Handler: cancelScheduleHandler, Sudo: true},
		},
	})
}
