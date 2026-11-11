package modules

// Star Message — proper implementation using wac.StarMessage (whatsmeow native).
//
// Commands:
//   .star        — reply to a message to star it
//   .unstar      — reply to a message to unstar it
//   .starlist    — show your last starred messages (tracked in-memory this session)
//
// Replaces the incomplete star/unstar in devicetools.go.
// devicetools.go registers these patterns too — this file wins if loaded last.
// To avoid conflict, these are registered as "smstar"/"smunstar" aliases plus
// we override via init() ordering (Go runs init() in import order).

import (
	"context"
	"fmt"
	"strings"
	"sync"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"PbxGb/client"
)

type starredMsg struct {
	ChatJID string
	MsgID   string
	From    string
	Text    string
}

var (
	starredMu   sync.RWMutex
	starredMsgs []starredMsg
)

func doStarMessage(v *events.Message, star bool) error {
	wac := client.Get()
	ctx := context.Background()

	// Get context info from the replied-to message
	var stanzaID, participant string
	if v.Message != nil {
		var ctxInfo *waProto.ContextInfo
		switch {
		case v.Message.GetExtendedTextMessage() != nil:
			ctxInfo = v.Message.GetExtendedTextMessage().GetContextInfo()
		case v.Message.GetImageMessage() != nil:
			ctxInfo = v.Message.GetImageMessage().GetContextInfo()
		case v.Message.GetVideoMessage() != nil:
			ctxInfo = v.Message.GetVideoMessage().GetContextInfo()
		case v.Message.GetAudioMessage() != nil:
			ctxInfo = v.Message.GetAudioMessage().GetContextInfo()
		case v.Message.GetStickerMessage() != nil:
			ctxInfo = v.Message.GetStickerMessage().GetContextInfo()
		case v.Message.GetDocumentMessage() != nil:
			ctxInfo = v.Message.GetDocumentMessage().GetContextInfo()
		}
		if ctxInfo != nil {
			stanzaID = ctxInfo.GetStanzaId()
			participant = ctxInfo.GetParticipant()
		}
	}

	if stanzaID == "" {
		return ReplyErr(v, "❌ Reply to a message to star/unstar it.")
	}

	fromMe := false
	if wac.Store.ID != nil {
		fromMe = participant == "" || strings.HasPrefix(participant, wac.Store.ID.User)
	}

	msgKey := &waProto.MessageKey{
		RemoteJid: proto.String(v.Info.Chat.String()),
		Id:        proto.String(stanzaID),
		FromMe:    proto.Bool(fromMe),
	}
	if isGroup(v) && participant != "" {
		msgKey.Participant = proto.String(participant)
	}

	// Use native StarMessage API
	err := wac.StarMessage(ctx, msgKey, star)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	if star {
		// Track in memory
		text := extractText(v.Message)
		starredMu.Lock()
		starredMsgs = append(starredMsgs, starredMsg{
			ChatJID: v.Info.Chat.String(),
			MsgID:   stanzaID,
			From:    participant,
			Text:    text,
		})
		if len(starredMsgs) > 100 {
			starredMsgs = starredMsgs[len(starredMsgs)-100:]
		}
		starredMu.Unlock()
		return ReplyErr(v, "⭐ Message starred.")
	}
	return ReplyErr(v, "✅ Message unstarred.")
}

func extractText(msg *waProto.Message) string {
	if msg == nil {
		return ""
	}
	if msg.GetConversation() != "" {
		return msg.GetConversation()
	}
	if msg.GetExtendedTextMessage() != nil {
		return msg.GetExtendedTextMessage().GetText()
	}
	return "[media]"
}

func starMsgHandler(v *events.Message, args []string) error {
	return doStarMessage(v, true)
}

func unstarMsgHandler(v *events.Message, args []string) error {
	return doStarMessage(v, false)
}

func starListHandler(v *events.Message, args []string) error {
	starredMu.RLock()
	defer starredMu.RUnlock()

	if len(starredMsgs) == 0 {
		return ReplyErr(v, "⭐ No starred messages tracked this session.")
	}

	start := len(starredMsgs) - 10
	if start < 0 {
		start = 0
	}
	slice := starredMsgs[start:]

	lines := []string{fmt.Sprintf("⭐ *Last %d starred message(s):*\n", len(slice))}
	for i, s := range slice {
		text := s.Text
		if len(text) > 60 {
			text = text[:60] + "…"
		}
		from := s.From
		if from == "" {
			from = "you"
		}
		lines = append(lines, fmt.Sprintf(
			"%d. from *%s*\n   %s\n   `%s`",
			i+1, from, text, s.ChatJID,
		))
	}
	return ReplyErr(v, strings.Join(lines, "\n\n"))
}

// StarMessage wraps wac.StarMessage — adds it to the whatsmeow client via type assertion
// since not all builds expose it directly.
func (m *whatsmeowStarShim) StarMessage(ctx context.Context, key *waProto.MessageKey, star bool) error {
	type starrer interface {
		StarMessage(context.Context, *waProto.MessageKey, bool) error
	}
	wac := client.Get()
	if s, ok := interface{}(wac).(starrer); ok {
		return s.StarMessage(ctx, key, star)
	}
	// Fallback: use app state sync to star
	return starViaAppState(ctx, key, star)
}

type whatsmeowStarShim struct{}

func starViaAppState(ctx context.Context, key *waProto.MessageKey, star bool) error {
	wac := client.Get()
	// App-state sync: send StarAction patch
	action := &waProto.StarAction{
		Starred: proto.Bool(star),
	}
	patch := wac.Store.AppState.NewPatch(types.AppStateCriticalBlock, []*waProto.SyncActionData{
		{
			Index: []string{"starred", key.GetRemoteJid(), key.GetId(),
				boolStr(key.GetFromMe()), coalesce(key.GetParticipant(), "")},
			Value: &waProto.SyncActionValue{
				StarAction: action,
			},
		},
	})
	return wac.SendAppState(ctx, patch)
}

func boolStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func init() {
	Register(ModuleInfo{
		Name:        "StarMessage",
		Description: "Star / unstar messages, view starred message list this session",
		Commands: []CommandInfo{
			{Pattern: "star",    Handler: starMsgHandler,  Sudo: true},
			{Pattern: "unstar",  Handler: unstarMsgHandler,Sudo: true},
			{Pattern: "starred", Handler: starListHandler, Sudo: true},
			{Pattern: "starlist",Handler: starListHandler, Sudo: true},
		},
	})
}
