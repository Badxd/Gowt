package modules

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// .stats        — show message stats for current group
// .topmembers   — top 10 most active members
// .mystats      — your own stats in this group
// .resetstats   — reset stats (sudo only)

type memberStat struct {
	JID      types.JID
	Name     string
	Messages int
	Words    int
	Stickers int
	Images   int
	Last     time.Time
}

type groupStats struct {
	Members map[string]*memberStat
	Total   int
	Started time.Time
}

var (
	statsMu   sync.RWMutex
	statsData = make(map[string]*groupStats) // chatJID → stats
)

// TrackMessage is called from the global message handler for every incoming message
func TrackMessage(v *events.Message) {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return // only groups
	}

	chatID := v.Info.Chat.String()
	senderID := v.Info.Sender.String()

	statsMu.Lock()
	defer statsMu.Unlock()

	gs, ok := statsData[chatID]
	if !ok {
		gs = &groupStats{
			Members: make(map[string]*memberStat),
			Started: time.Now(),
		}
		statsData[chatID] = gs
	}

	ms, ok := gs.Members[senderID]
	if !ok {
		ms = &memberStat{JID: v.Info.Sender}
		gs.Members[senderID] = ms
	}

	ms.Messages++
	gs.Total++
	ms.Last = v.Info.Timestamp

	text := ""
	if t := v.Message.GetConversation(); t != "" {
		text = t
	} else if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		text = ext.GetText()
	}
	if text != "" {
		ms.Words += len(strings.Fields(text))
	}

	if v.Message.GetStickerMessage() != nil {
		ms.Stickers++
	}
	if v.Message.GetImageMessage() != nil {
		ms.Images++
	}
	if v.Info.PushName != "" {
		ms.Name = v.Info.PushName
	}
}

func statsHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ Stats only work in groups")
	}

	chatID := v.Info.Chat.String()
	statsMu.RLock()
	gs := statsData[chatID]
	statsMu.RUnlock()

	if gs == nil || gs.Total == 0 {
		return ReplyErr(v, "📊 No stats recorded yet. Start chatting!")
	}

	since := time.Since(gs.Started)
	hours := int(since.Hours())
	days := hours / 24

	return ReplyErr(v, fmt.Sprintf(
		"📊 *Group Stats*\n\n💬 *Total Messages:* %d\n👥 *Active Members:* %d\n📅 *Tracking Since:* %d days ago\n⏱️ *Hours Active:* %d\n\nUse *.topmembers* to see top chatters.",
		gs.Total, len(gs.Members), days, hours))
}

func topMembersHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command only works in groups")
	}

	chatID := v.Info.Chat.String()
	statsMu.RLock()
	gs := statsData[chatID]
	statsMu.RUnlock()

	if gs == nil || len(gs.Members) == 0 {
		return ReplyErr(v, "📊 No stats yet. Start chatting!")
	}

	members := make([]*memberStat, 0, len(gs.Members))
	for _, m := range gs.Members {
		members = append(members, m)
	}
	sort.Slice(members, func(i, j int) bool {
		return members[i].Messages > members[j].Messages
	})

	var sb strings.Builder
	sb.WriteString("🏆 *Top Members*\n\n")

	medals := []string{"🥇", "🥈", "🥉"}
	limit := 10
	if len(members) < limit {
		limit = len(members)
	}
	for i := 0; i < limit; i++ {
		m := members[i]
		rank := fmt.Sprintf("%d.", i+1)
		if i < 3 {
			rank = medals[i]
		}
		name := m.Name
		if name == "" {
			name = "@" + m.JID.User
		}
		sb.WriteString(fmt.Sprintf("%s *%s* — %d msgs\n", rank, name, m.Messages))
	}
	return ReplyErr(v, sb.String())
}

func myStatsHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command only works in groups")
	}

	chatID := v.Info.Chat.String()
	senderID := v.Info.Sender.String()

	statsMu.RLock()
	gs := statsData[chatID]
	statsMu.RUnlock()

	if gs == nil {
		return ReplyErr(v, "📊 No stats yet for this group.")
	}

	ms := gs.Members[senderID]
	if ms == nil {
		return ReplyErr(v, "📊 No stats recorded for you yet.")
	}

	members := make([]*memberStat, 0, len(gs.Members))
	for _, m := range gs.Members {
		members = append(members, m)
	}
	sort.Slice(members, func(i, j int) bool {
		return members[i].Messages > members[j].Messages
	})
	rank := 0
	for i, m := range members {
		if m.JID.String() == senderID {
			rank = i + 1
			break
		}
	}

	percent := float64(ms.Messages) * 100 / float64(gs.Total)
	return ReplyErr(v, fmt.Sprintf(
		"📊 *Your Stats*\n\n💬 *Messages:* %d (%.1f%% of total)\n📝 *Words:* %d\n🖼️ *Images:* %d\n🎭 *Stickers:* %d\n🏆 *Rank:* #%d of %d",
		ms.Messages, percent, ms.Words, ms.Images, ms.Stickers, rank, len(gs.Members),
	))
}

func resetStatsHandler(v *events.Message, args []string) error {
	chatID := v.Info.Chat.String()
	statsMu.Lock()
	delete(statsData, chatID)
	statsMu.Unlock()
	return ReplyErr(v, "✅ Stats reset for this chat.")
}

func init() {
	Register(ModuleInfo{
		Name:        "GroupStats",
		Description: "Track and display group message statistics",
		Commands: []CommandInfo{
			{Pattern: "stats", Handler: statsHandler, Sudo: true},
			{Pattern: "topmembers", Handler: topMembersHandler, Sudo: true},
			{Pattern: "mystats", Handler: myStatsHandler, Sudo: true},
			{Pattern: "resetstats", Handler: resetStatsHandler, Sudo: true},
		},
	})
}
