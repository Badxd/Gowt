package modules

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// Message delete / purge commands.
//
// .del / .delete     — delete the quoted/replied-to message
// .purge <n>         — delete last n messages sent by the bot (tracked internally)

// msgTracker tracks recent message IDs sent by the bot per chat for purge.
var (
	msgTrackerMu   sync.Mutex
	msgTrackerMap  = make(map[string][]string) // chatJID -> []msgID (most recent last)
	msgTrackerMax  = 50                        // keep last 50 per chat
)

// TrackSentMessage should be called after every SendMessage to track bot's own messages.
func TrackSentMessage(chatJID, msgID string) {
	msgTrackerMu.Lock()
	defer msgTrackerMu.Unlock()
	list := msgTrackerMap[chatJID]
	list = append(list, msgID)
	if len(list) > msgTrackerMax {
		list = list[len(list)-msgTrackerMax:]
	}
	msgTrackerMap[chatJID] = list
}

func delHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	// Try to get quoted message ID
	ctxInfo := v.Message.GetExtendedTextMessage().GetContextInfo()
	if ctxInfo == nil || ctxInfo.GetStanzaId() == "" {
		return ReplyErr(v, "❌ Reply to a message to delete it.")
	}

	msgID := ctxInfo.GetStanzaId()
	fromMe := ctxInfo.GetParticipant() == "" // rough check: if participant empty, may be own msg

	ctx, cancel := newContext(10)
	defer cancel()

	err := wac.RevokeMessage(ctx, v.Info.Chat, fromMe, msgID, "", "")
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Could not delete message: %v", err))
	}

	// Also delete the .del command message itself
	time.Sleep(300 * time.Millisecond)
	ctx2, cancel2 := newContext(10)
	defer cancel2()
	_ = wac.RevokeMessage(ctx2, v.Info.Chat, true, v.Info.ID, "", "")

	return nil
}

func purgeHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	n := 10 // default
	if len(args) > 0 {
		if parsed, err := strconv.Atoi(args[0]); err == nil && parsed > 0 {
			n = parsed
		}
	}
	if n > 50 {
		n = 50
	}

	chat := v.Info.Chat.String()

	msgTrackerMu.Lock()
	list := msgTrackerMap[chat]
	// take last n
	if len(list) > n {
		list = list[len(list)-n:]
	}
	// clear from tracker
	if len(list) > 0 {
		remaining := msgTrackerMap[chat]
		msgTrackerMap[chat] = remaining[:len(remaining)-len(list)]
	}
	msgTrackerMu.Unlock()

	if len(list) == 0 {
		return ReplyErr(v, "ℹ️ No tracked bot messages to purge in this chat.")
	}

	deleted := 0
	for _, msgID := range list {
		ctx, cancel := newContext(8)
		err := wac.RevokeMessage(ctx, v.Info.Chat, true, msgID, "", "")
		cancel()
		if err == nil {
			deleted++
		}
		time.Sleep(150 * time.Millisecond)
	}

	// delete the purge command itself
	ctx, cancel := newContext(8)
	defer cancel()
	_ = wac.RevokeMessage(ctx, v.Info.Chat, true, v.Info.ID, "", "")

	return ReplyErr(v, fmt.Sprintf("🗑️ Purged *%d* bot message(s).", deleted))
}

func init() {
	Register(ModuleInfo{
		Name:        "Purge",
		Description: "Delete messages: single delete or bulk purge of bot messages",
		Commands: []CommandInfo{
			{Pattern: "delete",  Handler: delHandler,  Sudo: true},
			{Pattern: "delmsg",  Handler: delHandler,  Sudo: true},
			{Pattern: "purge", Handler: purgeHandler, Sudo: true},
		},
	})
}
