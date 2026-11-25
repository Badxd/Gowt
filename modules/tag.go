package modules

import (
	"fmt"
	"strings"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// .tagall [message]  — mention all group members with an optional message
// .tag @user1 @user2 [message] — mention specific users (same as typing @mention but via bot)
// .hidetag [message] — mention all without showing the @s visually (zero-width space trick)

func tagChatAllHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	ctx, cancel := newContext(20)
	defer cancel()

	groupInfo, err := wac.GetGroupInfo(ctx, v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Could not fetch group info: %v", err))
	}

	text := "@everyone"
	if len(args) > 0 {
		text = strings.Join(args, " ")
	}

	var mentions []string
	var mentionLines []string
	for _, p := range groupInfo.Participants {
		jidStr := p.JID.String()
		mentions = append(mentions, jidStr)
		mentionLines = append(mentionLines, "@"+p.JID.User)
	}

	fullText := text + "\n" + strings.Join(mentionLines, " ")

	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(fullText),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: mentions,
			},
		},
	}

	_, sendErr := wac.SendMessage(ctx, v.Info.Chat, msg)
	if sendErr != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Send failed: %v", sendErr))
	}
	return nil
}

func hideTagHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	ctx, cancel := newContext(20)
	defer cancel()

	groupInfo, err := wac.GetGroupInfo(ctx, v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Could not fetch group info: %v", err))
	}

	msgText := "📢 Attention"
	if len(args) > 0 {
		msgText = strings.Join(args, " ")
	}

	var mentions []string
	// Use zero-width space so mentions are invisible in text but still notify
	var hiddenMentions strings.Builder
	for _, p := range groupInfo.Participants {
		jidStr := p.JID.String()
		mentions = append(mentions, jidStr)
		// Each @tag is hidden between zero-width spaces
		hiddenMentions.WriteString("​@" + p.JID.User)
	}

	fullText := msgText + hiddenMentions.String()

	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(fullText),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: mentions,
			},
		},
	}

	_, sendErr := wac.SendMessage(ctx, v.Info.Chat, msg)
	if sendErr != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Send failed: %v", sendErr))
	}
	return nil
}

func tagAdminsHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	ctx, cancel := newContext(20)
	defer cancel()

	groupInfo, err := wac.GetGroupInfo(ctx, v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Could not fetch group info: %v", err))
	}

	text := "📢 Admins"
	if len(args) > 0 {
		text = strings.Join(args, " ")
	}

	var mentions []string
	var mentionLines []string
	for _, p := range groupInfo.Participants {
		if p.IsAdmin || p.IsSuperAdmin {
			mentions = append(mentions, p.JID.String())
			mentionLines = append(mentionLines, "@"+p.JID.User)
		}
	}

	if len(mentions) == 0 {
		return ReplyErr(v, "ℹ️ No admins found in this group.")
	}

	fullText := text + "\n" + strings.Join(mentionLines, " ")

	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(fullText),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: mentions,
			},
		},
	}

	_, sendErr := wac.SendMessage(ctx, v.Info.Chat, msg)
	if sendErr != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Send failed: %v", sendErr))
	}
	return nil
}

func init() {
	Register(ModuleInfo{
		Name:        "Tag",
		Description: "Tag all members, admins, or silently notify all in a group",
		Commands: []CommandInfo{
			{Pattern: "tag", Handler: tagChatAllHandler, Sudo: true},
			{Pattern: "hidetag", Handler: hideTagHandler, Sudo: true},
			{Pattern: "tagadmins", Handler: tagAdminsHandler, Sudo: true},
		},
	})
}
