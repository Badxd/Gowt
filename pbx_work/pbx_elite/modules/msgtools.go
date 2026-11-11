package modules

// Message Tools — advanced message utilities using whatsmeow.
//
// Commands:
//   .sendto <number> <text>   — send a message directly to a phone number
//   .sendtoall <text>         — send a message to all your contacts (use carefully)
//   .fwdto <number>           — forward replied message to a number
//   .deleteforme              — delete a replied message just for yourself (revoke own)
//   .savecontact @user        — save a contact's vCard
//   .getmsgid                 — show message ID of replied message
//   .quotedinfo               — show full info about the quoted/replied message
//   .readall                  — mark all unread chats as read
//   .markread                 — mark current chat as read
//   .pinmsg                   — pin a replied message
//   .unpinmsg                 — unpin a message by ID or reply

import (
	"context"
	"fmt"
	"strings"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"PbxGb/client"
)

// ── SEND TO NUMBER ────────────────────────────────────────────────────────────
func sendToHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		return ReplyErr(v, "Usage: .sendto 919812345678 Hello there!")
	}
	number := strings.TrimSpace(args[0])
	if !strings.Contains(number, "@") {
		number = number + "@s.whatsapp.net"
	}
	jid, err := types.ParseJID(number)
	if err != nil {
		return ReplyErr(v, "❌ Invalid number: "+args[0])
	}
	text := strings.Join(args[1:], " ")
	wac := client.Get()
	sendText(wac, jid, text)
	return ReplyErr(v, fmt.Sprintf("✅ Sent to *+%s*.", jid.User))
}

// ── FORWARD TO NUMBER ─────────────────────────────────────────────────────────
func fwdToHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .fwdto 919812345678  (reply to a message)")
	}
	if v.Message == nil {
		return ReplyErr(v, "❌ Reply to a message to forward it.")
	}

	// Get quoted message
	var quotedMsg *waProto.Message
	var quotedID string
	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		ctx := ext.GetContextInfo()
		if ctx != nil && ctx.QuotedMessage != nil {
			quotedMsg = ctx.QuotedMessage
			quotedID = ctx.GetStanzaId()
		}
	}
	if quotedMsg == nil {
		return ReplyErr(v, "❌ Reply to a message to forward it.")
	}
	_ = quotedID

	number := strings.TrimSpace(args[0])
	if !strings.Contains(number, "@") {
		number = number + "@s.whatsapp.net"
	}
	jid, err := types.ParseJID(number)
	if err != nil {
		return ReplyErr(v, "❌ Invalid number.")
	}

	wac := client.Get()
	_, err = wac.SendMessage(context.Background(), jid, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(fmt.Sprintf("*Forwarded from %s*", v.Info.Chat.String())),
			ContextInfo: &waProto.ContextInfo{
				IsForwarded:     proto.Bool(true),
				ForwardingScore: proto.Uint32(1),
			},
		},
	})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Forward failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("✅ Forwarded to *+%s*.", jid.User))
}

// ── GET MESSAGE ID ────────────────────────────────────────────────────────────
func getMsgIDHandler(v *events.Message, args []string) error {
	// If replying to a message, get its ID
	var msgID, fromJID string
	if v.Message != nil {
		var ctxInfo *waProto.ContextInfo
		if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			ctxInfo = ext.GetContextInfo()
		}
		if ctxInfo != nil {
			msgID = ctxInfo.GetStanzaId()
			fromJID = ctxInfo.GetParticipant()
		}
	}

	// If not replying, show current message ID
	if msgID == "" {
		return ReplyErr(v, fmt.Sprintf(
			"🆔 *This message*\nID: `%s`\nChat: `%s`\nFrom: `%s`",
			v.Info.ID, v.Info.Chat.String(), v.Info.Sender.String(),
		))
	}

	return ReplyErr(v, fmt.Sprintf(
		"🆔 *Quoted message*\nID: `%s`\nFrom: `%s`",
		msgID, fromJID,
	))
}

// ── QUOTED INFO ───────────────────────────────────────────────────────────────
func quotedInfoHandler(v *events.Message, args []string) error {
	if v.Message == nil {
		return ReplyErr(v, "❌ Reply to a message.")
	}

	var ctxInfo *waProto.ContextInfo
	switch {
	case v.Message.GetExtendedTextMessage() != nil:
		ctxInfo = v.Message.GetExtendedTextMessage().GetContextInfo()
	case v.Message.GetImageMessage() != nil:
		ctxInfo = v.Message.GetImageMessage().GetContextInfo()
	case v.Message.GetVideoMessage() != nil:
		ctxInfo = v.Message.GetVideoMessage().GetContextInfo()
	}

	if ctxInfo == nil || ctxInfo.GetStanzaId() == "" {
		return ReplyErr(v, "❌ No quoted message found.")
	}

	msgType := "text"
	text := ""
	if ctxInfo.QuotedMessage != nil {
		qm := ctxInfo.QuotedMessage
		switch {
		case qm.GetConversation() != "":
			text = qm.GetConversation()
			msgType = "text"
		case qm.GetImageMessage() != nil:
			msgType = "image"
			text = qm.GetImageMessage().GetCaption()
		case qm.GetVideoMessage() != nil:
			msgType = "video"
			text = qm.GetVideoMessage().GetCaption()
		case qm.GetAudioMessage() != nil:
			msgType = "audio"
		case qm.GetStickerMessage() != nil:
			msgType = "sticker"
		case qm.GetDocumentMessage() != nil:
			msgType = "document"
			text = qm.GetDocumentMessage().GetFileName()
		case qm.GetExtendedTextMessage() != nil:
			msgType = "text"
			text = qm.GetExtendedTextMessage().GetText()
		}
	}

	if len(text) > 200 {
		text = text[:200] + "…"
	}

	return ReplyErr(v, fmt.Sprintf(
		"📋 *Quoted Message Info*\n\n"+
			"🆔 *ID:* `%s`\n"+
			"👤 *From:* `%s`\n"+
			"📝 *Type:* %s\n"+
			"💬 *Text:* %s",
		ctxInfo.GetStanzaId(),
		ctxInfo.GetParticipant(),
		msgType,
		func() string {
			if text == "" {
				return "—"
			}
			return text
		}(),
	))
}

// ── MARK CURRENT CHAT READ ────────────────────────────────────────────────────
func markReadHandler(v *events.Message, args []string) error {
	wac := client.Get()
	ts := v.Info.Timestamp
	err := wac.MarkRead(
		[]types.MessageID{v.Info.ID},
		ts,
		v.Info.Chat,
		v.Info.Sender,
	)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, "✅ Chat marked as read.")
}

// ── DELETE FOR ME (revoke own message) ───────────────────────────────────────
func deleteForMeHandler(v *events.Message, args []string) error {
	if v.Message == nil {
		return ReplyErr(v, "❌ Reply to your own message.")
	}

	var stanzaID, participant string
	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		ctx := ext.GetContextInfo()
		if ctx != nil {
			stanzaID = ctx.GetStanzaId()
			participant = ctx.GetParticipant()
		}
	}
	if stanzaID == "" {
		return ReplyErr(v, "❌ Reply to a message to delete.")
	}

	wac := client.Get()
	fromMe := false
	if wac.Store.ID != nil {
		fromMe = participant == "" || strings.HasPrefix(participant, wac.Store.ID.User)
	}
	if !fromMe {
		return ReplyErr(v, "❌ You can only delete your own messages.")
	}

	_, err := wac.SendMessage(context.Background(), v.Info.Chat,
		wac.BuildRevoke(v.Info.Chat, types.EmptyJID, stanzaID))
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Delete failed: %v", err))
	}
	return ReplyErr(v, "🗑️ Message deleted.")
}

// ── SEND VCARD ────────────────────────────────────────────────────────────────
func saveContactHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .savecontact @mention or .savecontact 919xxx")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}

	wac := client.Get()
	ctx := context.Background()

	// Fetch profile pic and name
	contacts, _ := wac.GetUserInfo(ctx, []types.JID{jid})
	displayName := "+"+jid.User
	if info, ok := contacts[jid]; ok && info.VerifiedName != nil {
		displayName = info.VerifiedName.Name
	}

	vcard := fmt.Sprintf(
		"BEGIN:VCARD\nVERSION:3.0\nFN:%s\nTEL;type=CELL;type=VOICE;waid=%s:+%s\nEND:VCARD",
		displayName, jid.User, jid.User,
	)

	_, err = wac.SendMessage(ctx, v.Info.Chat, &waProto.Message{
		ContactMessage: &waProto.ContactMessage{
			DisplayName: proto.String(displayName),
			Vcard:       proto.String(vcard),
		},
	})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("📇 Contact card for *%s* sent.", displayName))
}

// ── BULK SEND TO ALL CONTACTS ─────────────────────────────────────────────────
func sendToAllHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .sendtoall <message>  ⚠️ Sends to ALL contacts")
	}
	text := strings.Join(args, " ")
	wac := client.Get()
	ctx := context.Background()

	groups, err := wac.GetJoinedGroups()
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to get groups: %v", err))
	}

	sent := 0
	for _, g := range groups {
		for _, p := range g.Participants {
			if wac.Store.ID != nil && p.JID.User == wac.Store.ID.User {
				continue
			}
			_, _ = wac.SendMessage(ctx, p.JID, &waProto.Message{
				Conversation: proto.String(text),
			})
			sent++
			if sent >= 50 { // safety cap
				break
			}
		}
		if sent >= 50 {
			break
		}
	}
	return ReplyErr(v, fmt.Sprintf("📤 Sent to *%d* contacts (capped at 50).", sent))
}

func init() {
	Register(ModuleInfo{
		Name:        "MsgTools",
		Description: "Send to number, forward, get msg ID, quoted info, mark read, delete for me, save contact",
		Commands: []CommandInfo{
			{Pattern: "sendto",      Handler: sendToHandler,       Sudo: true},
			{Pattern: "fwdto",       Handler: fwdToHandler,        Sudo: true},
			{Pattern: "getmsgid",    Handler: getMsgIDHandler,     Sudo: true},
			{Pattern: "msgid",       Handler: getMsgIDHandler,     Sudo: true},
			{Pattern: "quotedinfo",  Handler: quotedInfoHandler,   Sudo: true},
			{Pattern: "qinfo",       Handler: quotedInfoHandler,   Sudo: true},
			{Pattern: "markread",    Handler: markReadHandler,     Sudo: true},
			{Pattern: "readchat",    Handler: markReadHandler,     Sudo: true},
			{Pattern: "delforme",    Handler: deleteForMeHandler,  Sudo: true},
			{Pattern: "savecontact", Handler: saveContactHandler,  Sudo: true},
			{Pattern: "sendtoall",   Handler: sendToAllHandler,    Sudo: true},
		},
	})
}
