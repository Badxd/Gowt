package modules

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"PbxGb/client"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// PBX Elite — Clean Module
//
// .clean <count>    — delete last <count> tracked bot messages (default 10)
// .ikallclean       — reply to a message; delete that msg + everything after it that bot sent

// ─── Message tracker ──────────────────────────────────────────────────────
// chatJID → ordered list of (msgID, senderJID)
type trackedMsg struct {
	ID        string
	SenderJID string
	Timestamp time.Time
}

var (
	trackerMu   sync.Mutex
	msgTracker  = make(map[string][]trackedMsg) // chatJID → msgs (oldest first)
	maxTracked  = 500
)

// TrackSentMessage — call this after every wac.SendMessage to log what bot sent
func TrackSentMessage(chatJID, msgID string) {
	wac := client.Get()
	senderJID := ""
	if wac != nil && wac.Store.ID != nil {
		senderJID = wac.Store.ID.User + "@" + types.DefaultUserServer
	}
	trackerMu.Lock()
	defer trackerMu.Unlock()
	list := msgTracker[chatJID]
	list = append(list, trackedMsg{ID: msgID, SenderJID: senderJID, Timestamp: time.Now()})
	if len(list) > maxTracked {
		list = list[len(list)-maxTracked:]
	}
	msgTracker[chatJID] = list
}

// TrackIncomingMessage — also track any message by its ID and sender for purge
func TrackIncomingMessage(chatJID, msgID, senderJID string) {
	trackerMu.Lock()
	defer trackerMu.Unlock()
	list := msgTracker[chatJID]
	list = append(list, trackedMsg{ID: msgID, SenderJID: senderJID, Timestamp: time.Now()})
	if len(list) > maxTracked {
		list = list[len(list)-maxTracked:]
	}
	msgTracker[chatJID] = list
}

func revokeMsg(chatJID types.JID, msgID string) {
	wac := client.Get()
	if wac == nil {
		return
	}
	revokeProto := wac.BuildRevoke(chatJID, types.EmptyJID, msgID)
	_, _ = wac.SendMessage(context.Background(), chatJID, revokeProto)
	time.Sleep(120 * time.Millisecond)
}

// cleanHandler — .clean <count>: delete last N tracked bot messages
func cleanHandler(v *events.Message, args []string) error {
	wac := client.Get()
	if wac == nil || wac.Store.ID == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	count := 10
	if len(args) > 0 {
		fmt.Sscanf(args[0], "%d", &count)
	}
	if count < 1 {
		count = 1
	}
	if count > 100 {
		count = 100
	}

	chatJID := v.Info.Chat
	chatStr := chatJID.String()

	// Always revoke the .clean command itself
	revokeMsg(chatJID, v.Info.ID)

	botUser := ""
	if wac.Store.ID != nil {
		botUser = wac.Store.ID.User
	}

	trackerMu.Lock()
	list := msgTracker[chatStr]
	// collect last `count` bot-sent entries only, newest first
	var toDelete []string
	for i := len(list) - 1; i >= 0 && len(toDelete) < count; i-- {
		m := list[i]
		// Match only messages sent by the bot itself
		senderUser := m.SenderJID
		if idx := len(senderUser) - len("@s.whatsapp.net"); idx > 0 && senderUser[idx:] == "@s.whatsapp.net" {
			senderUser = senderUser[:idx]
		}
		if botUser != "" && senderUser != botUser {
			continue
		}
		toDelete = append(toDelete, m.ID)
	}
	// remove deleted from tracker
	if len(toDelete) > 0 {
		keep := make([]trackedMsg, 0, len(list))
		del := make(map[string]bool)
		for _, id := range toDelete {
			del[id] = true
		}
		for _, m := range list {
			if !del[m.ID] {
				keep = append(keep, m)
			}
		}
		msgTracker[chatStr] = keep
	}
	trackerMu.Unlock()

	deleted := 0
	for _, id := range toDelete {
		revokeMsg(chatJID, id)
		deleted++
	}

	if deleted == 0 {
		confirm := "🧹 No tracked bot messages to delete.\n💡 Bot messages are tracked since last restart."
		resp, _ := wac.SendMessage(context.Background(), chatJID, &waProto.Message{Conversation: &confirm})
		TrackSentMessage(chatStr, resp.ID)
		return nil
	}

	confirm := fmt.Sprintf("🧹 *Cleaned %d message(s).*", deleted)
	resp, _ := wac.SendMessage(context.Background(), chatJID, &waProto.Message{Conversation: &confirm})
	TrackSentMessage(chatStr, resp.ID)
	return nil
}

// ikallcleanHandler — .ikallclean: reply to a msg; delete it + all bot msgs after it
func ikallcleanHandler(v *events.Message, args []string) error {
	wac := client.Get()
	if wac == nil || wac.Store.ID == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	// Get the quoted message ID via ContextInfo
	var targetMsgID string
	if v.Message != nil {
		var ctxInfo interface{ GetStanzaId() string }
		if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			ctxInfo = ext.GetContextInfo()
		} else if img := v.Message.GetImageMessage(); img != nil {
			ctxInfo = img.GetContextInfo()
		} else if vid := v.Message.GetVideoMessage(); vid != nil {
			ctxInfo = vid.GetContextInfo()
		}
		if ctxInfo != nil {
			targetMsgID = ctxInfo.GetStanzaId()
		}
	}
	if targetMsgID == "" {
		return ReplyErr(v, "❌ Reply to a message to delete from that point onwards.")
	}

	chatJID := v.Info.Chat
	chatStr := chatJID.String()

	// Revoke the .ikallclean command itself
	revokeMsg(chatJID, v.Info.ID)

	trackerMu.Lock()
	list := msgTracker[chatStr]

	// Find position of target message
	targetIdx := -1
	for i, m := range list {
		if m.ID == targetMsgID {
			targetIdx = i
			break
		}
	}

	var toDelete []string
	if targetIdx >= 0 {
		// delete from targetIdx onwards
		for i := targetIdx; i < len(list); i++ {
			toDelete = append(toDelete, list[i].ID)
		}
		msgTracker[chatStr] = list[:targetIdx]
	} else {
		// target not in tracker — just delete everything tracked after session start
		for _, m := range list {
			toDelete = append(toDelete, m.ID)
		}
		msgTracker[chatStr] = nil
	}
	trackerMu.Unlock()

	// Also delete the target message itself if not in tracker
	found := false
	for _, id := range toDelete {
		if id == targetMsgID {
			found = true
			break
		}
	}
	if !found {
		revokeMsg(chatJID, targetMsgID)
	}

	deleted := 0
	for _, id := range toDelete {
		revokeMsg(chatJID, id)
		deleted++
	}

	confirm := fmt.Sprintf("🧹 *ikallclean done.* Deleted %d message(s) from that point.", deleted)
	msg := strings.TrimSpace(confirm)
	resp, _ := wac.SendMessage(context.Background(), chatJID, &waProto.Message{Conversation: &msg})
	TrackSentMessage(chatStr, resp.ID)
	return nil
}

func init() {
	Register(ModuleInfo{
		Name:        "Clean",
		Description: "Delete bot messages from chat (.clean) or from a point onwards (.ikallclean)",
		Commands: []CommandInfo{
			{Pattern: "clean", Handler: cleanHandler, Sudo: true},
			{Pattern: "ikallclean", Handler: ikallcleanHandler, Sudo: true},
		},
	})
}
