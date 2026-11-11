package modules

import (
	"fmt"
	"strings"
	"sync"
	"time"


	"go.mau.fi/whatsmeow/types/events"
)

// .mystats   — show your personal message stats
// .topusers  — top 10 active users in current group

type userStatEntry struct {
	JID         string
	PushName    string
	MsgCount    int
	CharCount   int
	MediaCount  int
	FirstSeen   time.Time
	LastSeen    time.Time
}

var (
	mystatsMu    sync.RWMutex
	userStatsMap = make(map[string]*userStatEntry) // "chatJID|userJID" -> entry
)

// TrackUserStats is exported and wired into HandleMessage
func TrackUserStats(v *events.Message) {
	if v.Info.IsFromMe {
		return
	}

	key := v.Info.Chat.String() + "|" + v.Info.Sender.String()

	text := ""
	if t := v.Message.GetConversation(); t != "" {
		text = t
	} else if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		text = ext.GetText()
	}

	isMedia := v.Message.GetImageMessage() != nil ||
		v.Message.GetVideoMessage() != nil ||
		v.Message.GetAudioMessage() != nil ||
		v.Message.GetDocumentMessage() != nil ||
		v.Message.GetStickerMessage() != nil

	mystatsMu.Lock()
	entry, ok := userStatsMap[key]
	if !ok {
		entry = &userStatEntry{
			JID:       v.Info.Sender.String(),
			PushName:  v.Info.PushName,
			FirstSeen: time.Now(),
		}
		userStatsMap[key] = entry
	}
	entry.MsgCount++
	entry.CharCount += len([]rune(text))
	if isMedia {
		entry.MediaCount++
	}
	entry.LastSeen = time.Now()
	if v.Info.PushName != "" {
		entry.PushName = v.Info.PushName
	}
	mystatsMu.Unlock()
}

func myPersonalStatsHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	// target: self by default, or mentioned user
	targetJID := wac.Store.ID
	targetName := wac.Store.PushName

	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		if ci := ext.GetContextInfo(); ci != nil {
			mentions := ci.GetMentionedJID()
			if len(mentions) > 0 {
				j, err := parseJID(mentions[0])
				if err == nil {
					targetJID = &j
					targetName = mentions[0]
				}
			}
		}
	}

	chat := v.Info.Chat.String()
	key := chat + "|" + targetJID.String()

	mystatsMu.RLock()
	entry, ok := userStatsMap[key]
	mystatsMu.RUnlock()

	if !ok {
		return ReplyErr(v, fmt.Sprintf("📊 No stats recorded for *%s* in this chat yet.", targetName))
	}

	name := entry.PushName
	if name == "" {
		parts := strings.SplitN(entry.JID, "@", 2)
		name = parts[0]
	}

	elapsed := time.Since(entry.FirstSeen)
	days := int(elapsed.Hours() / 24)
	daysStr := fmt.Sprintf("%d days", days)
	if days == 0 {
		daysStr = "today"
	} else if days == 1 {
		daysStr = "1 day"
	}

	msgsPerDay := 0.0
	if days > 0 {
		msgsPerDay = float64(entry.MsgCount) / float64(days)
	}

	avgLen := 0
	if entry.MsgCount > 0 {
		avgLen = entry.CharCount / entry.MsgCount
	}

	lastSeen := time.Since(entry.LastSeen).Round(time.Minute)

	lines := []string{
		fmt.Sprintf("📊 *Stats for %s*", name),
		"",
		fmt.Sprintf("💬 Messages: *%d*", entry.MsgCount),
		fmt.Sprintf("📝 Characters: *%d* (avg %d/msg)", entry.CharCount, avgLen),
		fmt.Sprintf("🖼️ Media: *%d*", entry.MediaCount),
		fmt.Sprintf("📈 Rate: *%.1f* msgs/day", msgsPerDay),
		fmt.Sprintf("🕒 Active for: *%s*", daysStr),
		fmt.Sprintf("👁️ Last seen: *%s ago*", lastSeen),
	}

	return ReplyErr(v, strings.Join(lines, "\n"))
}

func topUsersHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ Groups only.")
	}

	chat := v.Info.Chat.String()
	prefix := chat + "|"

	mystatsMu.RLock()
	var entries []*userStatEntry
	for k, e := range userStatsMap {
		if strings.HasPrefix(k, prefix) {
			cp := *e
			entries = append(entries, &cp)
		}
	}
	mystatsMu.RUnlock()

	if len(entries) == 0 {
		return ReplyErr(v, "📊 No stats recorded for this group yet.")
	}

	// sort by MsgCount descending (simple insertion sort, N is small)
	for i := 1; i < len(entries); i++ {
		cur := entries[i]
		j := i - 1
		for j >= 0 && entries[j].MsgCount < cur.MsgCount {
			entries[j+1] = entries[j]
			j--
		}
		entries[j+1] = cur
	}

	limit := 10
	if len(entries) < limit {
		limit = len(entries)
	}

	lines := []string{fmt.Sprintf("🏆 *Top %d Users in this Group*", limit), ""}
	medals := []string{"🥇", "🥈", "🥉"}

	for i := 0; i < limit; i++ {
		e := entries[i]
		name := e.PushName
		if name == "" {
			parts := strings.SplitN(e.JID, "@", 2)
			name = parts[0]
		}
		medal := "▫️"
		if i < len(medals) {
			medal = medals[i]
		}
		lines = append(lines, fmt.Sprintf("%s %d. *%s* — %d msgs", medal, i+1, name, e.MsgCount))
	}

	return ReplyErr(v, strings.Join(lines, "\n"))
}

func resetPersonalStatsHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()
	prefix := chat + "|"

	mystatsMu.Lock()
	count := 0
	for k := range userStatsMap {
		if strings.HasPrefix(k, prefix) {
			delete(userStatsMap, k)
			count++
		}
	}
	mystatsMu.Unlock()

	return ReplyErr(v, fmt.Sprintf("🗑️ Cleared stats for %d users in this chat.", count))
}

func init() {
	Register(ModuleInfo{
		Name:        "User Stats",
		Description: "Track and display per-user message statistics",
		Commands: []CommandInfo{
			{Pattern: "topusers", Handler: topUsersHandler, Sudo: true},
		},
	})
}
