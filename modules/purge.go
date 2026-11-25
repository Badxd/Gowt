package modules

import (
	"fmt"
	"strconv"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// Message delete / purge commands.
//
// .del / .delete  — delete the quoted/replied-to message
// .purge <n>      — delete last n messages sent by the bot
//
// NOTE: TrackSentMessage, TrackIncomingMessage, trackerMu, msgTracker are
// all defined in clean.go — this file just uses them.

func delHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	// Try to get quoted message ID
	ctxInfo := v.Message.GetExtendedTextMessage().GetContextInfo()
	if ctxInfo == nil || ctxInfo.GetStanzaID() == "" {
		return ReplyErr(v, "❌ Reply to a message to delete it.")
	}

	msgID := types.MessageID(ctxInfo.GetStanzaID())

	ctx, cancel := newContext(10)
	defer cancel()

	_, err := wac.RevokeMessage(ctx, v.Info.Chat, msgID)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Could not delete message: %v", err))
	}

	// Also delete the .del command message itself
	time.Sleep(300 * time.Millisecond)
	ctx2, cancel2 := newContext(10)
	defer cancel2()
	_, _ = wac.RevokeMessage(ctx2, v.Info.Chat, v.Info.ID)

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

	// Use clean.go's shared tracker — read bot's own messages only
	// TrackSentMessage stores SenderJID as "user@s.whatsapp.net"; match on User part.
	botUser := ""
	if wac.Store.ID != nil {
		botUser = wac.Store.ID.User
	}

	trackerMu.Lock()
	allMsgs := msgTracker[chat]

	// Filter to bot's own sent messages
	var botMsgs []trackedMsg
	var others []trackedMsg
	for _, m := range allMsgs {
		senderUser := m.SenderJID
		if idx := len(senderUser) - len("@s.whatsapp.net"); idx > 0 && senderUser[idx:] == "@s.whatsapp.net" {
			senderUser = senderUser[:idx]
		}
		if senderUser == botUser {
			botMsgs = append(botMsgs, m)
		} else {
			others = append(others, m)
		}
	}

	// Take last n from bot messages
	toDelete := botMsgs
	if len(toDelete) > n {
		toDelete = toDelete[len(toDelete)-n:]
		// Keep the ones not being deleted
		others = append(others, botMsgs[:len(botMsgs)-n]...)
	}
	// Rebuild tracker without the purged entries
	remaining := others
	msgTracker[chat] = remaining
	trackerMu.Unlock()

	if len(toDelete) == 0 {
		return ReplyErr(v, "ℹ️ No tracked bot messages to purge in this chat.")
	}

	deleted := 0
	for _, m := range toDelete {
		ctx, cancel := newContext(8)
		_, err := wac.RevokeMessage(ctx, v.Info.Chat, types.MessageID(m.ID))
		cancel()
		if err == nil {
			deleted++
		}
		time.Sleep(150 * time.Millisecond)
	}

	// delete the purge command itself
	ctx, cancel := newContext(8)
	defer cancel()
	_, _ = wac.RevokeMessage(ctx, v.Info.Chat, v.Info.ID)

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
