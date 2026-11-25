package modules

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// .statusview  — show all status updates tracked (from contacts who have viewed)
// .clearstatus — clear tracked status list
// status updates come in via events.Message with IsStatus == true
// this module tracks them passively via the hook

type statusEntry struct {
	SenderJID string
	PushName  string
	Text      string
	Type      string // text | image | video
	SeenAt    time.Time
}

var (
	statusMu      sync.RWMutex
	statusHistory []statusEntry
	maxStatusLog  = 100
)

// TrackStatus is exported and wired into HandleMessage for every message
func TrackStatus(v *events.Message) {
	if !v.Info.IsFromMe && v.Info.IsGroup {
		return // only DM statuses
	}

	// WhatsApp status updates have Chat == "status@broadcast"
	if v.Info.Chat.String() != "status@broadcast" {
		return
	}

	entry := statusEntry{
		SenderJID: v.Info.Sender.String(),
		PushName:  v.Info.PushName,
		SeenAt:    time.Now(),
	}

	if t := v.Message.GetConversation(); t != "" {
		entry.Text = t
		entry.Type = "text"
	} else if v.Message.GetImageMessage() != nil {
		caption := v.Message.GetImageMessage().GetCaption()
		entry.Text = caption
		entry.Type = "image"
	} else if v.Message.GetVideoMessage() != nil {
		caption := v.Message.GetVideoMessage().GetCaption()
		entry.Text = caption
		entry.Type = "video"
	} else if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		entry.Text = ext.GetText()
		entry.Type = "text"
	} else {
		entry.Type = "media"
	}

	statusMu.Lock()
	statusHistory = append([]statusEntry{entry}, statusHistory...) // newest first
	if len(statusHistory) > maxStatusLog {
		statusHistory = statusHistory[:maxStatusLog]
	}
	statusMu.Unlock()
}

func statusViewHandler(v *events.Message, args []string) error {
	statusMu.RLock()
	entries := make([]statusEntry, len(statusHistory))
	copy(entries, statusHistory)
	statusMu.RUnlock()

	if len(entries) == 0 {
		return ReplyErr(v, "📭 No status updates tracked yet.")
	}

	limit := 20
	if len(entries) < limit {
		limit = len(entries)
	}
	entries = entries[:limit]

	var lines []string
	lines = append(lines, fmt.Sprintf("📊 *Status Updates (%d)*", len(statusHistory)))
	lines = append(lines, "")

	for i, e := range entries {
		name := e.PushName
		if name == "" {
			// extract user from JID
			parts := strings.SplitN(e.SenderJID, "@", 2)
			name = parts[0]
		}
		elapsed := time.Since(e.SeenAt).Round(time.Minute)
		icon := "💬"
		switch e.Type {
		case "image":
			icon = "🖼️"
		case "video":
			icon = "🎬"
		case "media":
			icon = "📎"
		}

		line := fmt.Sprintf("%d. %s *%s* — %s ago", i+1, icon, name, elapsed)
		if e.Text != "" {
			preview := e.Text
			if len(preview) > 40 {
				preview = preview[:40] + "..."
			}
			line += "\n   _" + preview + "_"
		}
		lines = append(lines, line)
	}

	return ReplyErr(v, strings.Join(lines, "\n"))
}

func clearStatusHandler(v *events.Message, args []string) error {
	statusMu.Lock()
	count := len(statusHistory)
	statusHistory = nil
	statusMu.Unlock()

	return ReplyErr(v, fmt.Sprintf("🗑️ Cleared %d tracked status updates.", count))
}

func init() {
	Register(ModuleInfo{
		Name:        "Status Tracker",
		Description: "Track and view contact status updates",
		Commands: []CommandInfo{
			{Pattern: "statusview", Handler: statusViewHandler, Sudo: true},
			{Pattern: "sv", Handler: statusViewHandler, Sudo: true},
			{Pattern: "clearstatus", Handler: clearStatusHandler, Sudo: true},
		},
	})
}
