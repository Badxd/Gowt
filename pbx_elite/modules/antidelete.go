package modules

// PBX Elite — Anti Delete Module
// ✅ Detects deleted messages
// ✅ Saves & re-sends to owner's saved messages (self chat)
// ✅ Toggle on/off
// ✅ Works in DMs and groups

import (
	"context"
	"fmt"
	"sync"

	"PbxGb/client"
	"PbxGb/config"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	antiDeleteMu      sync.RWMutex
	antiDeleteEnabled = false
	// msgCache stores recent messages: msgID → content
	msgCache   = make(map[string]cachedMsg)
	msgCacheMu sync.RWMutex
)

type cachedMsg struct {
	SenderJID string
	ChatJID   string
	ChatName  string
	Text      string
}

// CacheMessage — stores every incoming message for anti-delete lookup
func CacheMessage(v *events.Message) {

	text := extractText(v)
	if text == "" {
		return
	}

	msgCacheMu.Lock()
	defer msgCacheMu.Unlock()

	// Keep cache size reasonable (max 500)
	if len(msgCache) > 500 {
		// Clear half
		count := 0
		for k := range msgCache {
			delete(msgCache, k)
			count++
			if count >= 250 {
				break
			}
		}
	}

	msgCache[v.Info.ID] = cachedMsg{
		SenderJID: v.Info.Sender.String(),
		ChatJID:   v.Info.Chat.String(),
		ChatName:  v.Info.PushName,
		Text:      text,
	}
}

// HandleAntiDelete — called from main.go on events.Receipt (revoke)
func HandleAntiDelete(v *events.Message) {

	antiDeleteMu.RLock()
	enabled := antiDeleteEnabled
	antiDeleteMu.RUnlock()

	if !enabled {
		return
	}

	// Check if this is a revoke (delete) message
	if v.Message == nil {
		return
	}

	proto := v.Message.GetProtocolMessage()
	if proto == nil {
		return
	}

	// Type 0 = REVOKE
	if proto.GetType() != waProto.ProtocolMessage_REVOKE {
		return
	}

	deletedKey := proto.GetKey()
	if deletedKey == nil {
		return
	}

	deletedID := deletedKey.GetID()

	// Look up cached message
	msgCacheMu.RLock()
	cached, found := msgCache[deletedID]
	msgCacheMu.RUnlock()

	if !found {
		return
	}

	wac := client.Get()
	if wac.Store.ID == nil {
		return
	}

	// Send to saved messages (self chat)
	ownerJIDStr := config.C.OwnerJID
	ownerUser := ownerJIDStr
	if len(ownerJIDStr) > 15 {
		// strip @s.whatsapp.net
		for i, c := range ownerJIDStr {
			if c == '@' {
				ownerUser = ownerJIDStr[:i]
				break
			}
		}
	}

	selfJID := types.NewJID(ownerUser, types.DefaultUserServer)

	notif := fmt.Sprintf(
		"╔══════════════════════╗\n"+
			"║  🗑️ *ANTI DELETE* 👻  ║\n"+
			"╚══════════════════════╝\n\n"+
			"*From:* %s\n"+
			"*Chat:* %s\n\n"+
			"*Deleted Message:*\n%s",
		cached.ChatName,
		cached.ChatJID,
		cached.Text,
	)

	_, _ = wac.SendMessage(
		context.Background(),
		selfJID,
		&waProto.Message{
			Conversation: &notif,
		},
	)
}

// .antidelete on/off
func antiDeleteToggleHandler(
	v *events.Message,
	args []string,
) error {

	if len(args) == 0 {
		antiDeleteMu.RLock()
		s := antiDeleteEnabled
		antiDeleteMu.RUnlock()

		state := "🟢 ON"
		if !s {
			state = "🔴 OFF"
		}
		return ReplyErr(v, fmt.Sprintf("🗑️ Anti Delete: *%s*", state))
	}

	switch args[0] {
	case "on":
		antiDeleteMu.Lock()
		antiDeleteEnabled = true
		antiDeleteMu.Unlock()
		return ReplyErr(v, "✅ Anti Delete *ON*\nDeleted messages saved to your Saved Messages.")

	case "off":
		antiDeleteMu.Lock()
		antiDeleteEnabled = false
		antiDeleteMu.Unlock()
		return ReplyErr(v, "🔴 Anti Delete *OFF*")
	}

	return ReplyErr(v, "❌ Use: .antidelete on / off")
}

// REGISTER
func init() {

	Register(ModuleInfo{
		Name:        "AntiDelete",
		Description: "Catch deleted messages",

		Commands: []CommandInfo{

			{
				Pattern: "antidelete",
				Handler: antiDeleteToggleHandler,
				Sudo:    true,
			},
		},
	})
}
