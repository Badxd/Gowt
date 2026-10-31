package modules

import (
	"context"
	"fmt"
	"strings"

	"PbxGb/client"
	"PbxGb/database"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// MaxWarnings is how many warnings a user can get before being auto-kicked.
const MaxWarnings = 3

// resolveMentionOrReplyTarget mirrors Knight Bot's warn.js target logic:
// prefer an @mention in the message, otherwise fall back to who's being
// replied to. Returns "" if neither is present.
func resolveMentionOrReplyTarget(v *events.Message) string {
	if v.Message != nil && v.Message.ExtendedTextMessage != nil {
		ctx := v.Message.ExtendedTextMessage.ContextInfo
		if ctx != nil && len(ctx.MentionedJID) > 0 {
			return ctx.MentionedJID[0]
		}
		if ctx != nil && ctx.Participant != nil && ctx.QuotedMessage != nil {
			return *ctx.Participant
		}
	}
	return ""
}

// argsAfterMention strips a leading mention from the args (Knight does the
// same: ".warn @user reason" — the mention itself isn't part of the reason).
func argsAfterMention(v *events.Message, args []string) []string {
	if v.Message != nil && v.Message.ExtendedTextMessage != nil {
		ctx := v.Message.ExtendedTextMessage.ContextInfo
		if ctx != nil && len(ctx.MentionedJID) > 0 && len(args) > 0 {
			return args[1:]
		}
	}
	return args
}

// isParticipantAdmin checks whether targetJID is an admin/superadmin in the group.
func isParticipantAdmin(groupJID, targetJID types.JID) bool {
	wac := client.Get()
	info, err := wac.GetGroupInfo(context.Background(), groupJID)
	if err != nil {
		return false
	}
	for _, p := range info.Participants {
		if p.JID.User == targetJID.User && (p.IsAdmin || p.IsSuperAdmin) {
			return true
		}
	}
	return false
}

// .warn @user <reason>  OR  reply + .warn <reason>
func warnHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}

	target := resolveMentionOrReplyTarget(v)
	if target == "" {
		return ReplyErr(v, "❌ Mention or reply to the user to warn!\nExample: .warn @user Breaking rules")
	}

	targetJID, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid user.")
	}

	if isParticipantAdmin(v.Info.Chat, targetJID) {
		return ReplyErr(v, "❌ Cannot warn an admin!")
	}

	reason := strings.Join(argsAfterMention(v, args), " ")
	if reason == "" {
		reason = "No reason specified"
	}

	count, err := database.AddWarning(v.Info.Chat.String(), target, reason)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Could not save warning: %v", err))
	}

	user := targetJID.User
	text := fmt.Sprintf("⚠️ *USER WARNING*\n\n👤 User: @%s\n📝 Reason: %s\n⚠️ Warnings: %d/%d\n\n",
		user, reason, count, MaxWarnings)

	if count >= MaxWarnings {
		text += "❌ User has reached maximum warnings and will be removed!"
	} else {
		text += "⚠️ Next warning will bring them closer to removal!"
	}

	wac := client.Get()
	_, _ = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: &text,
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: []string{target},
			},
		},
	})

	if count >= MaxWarnings {
		_, kickErr := wac.UpdateGroupParticipants(context.Background(), v.Info.Chat, []types.JID{targetJID}, "remove")
		if kickErr == nil {
			_ = database.ClearWarnings(v.Info.Chat.String(), target)
		} else {
			_ = ReplyErr(v, fmt.Sprintf("⚠️ Reached max warnings but couldn't remove them (am I admin?): %v", kickErr))
		}
	}

	return nil
}

// .resetwarn @user  OR  reply + .resetwarn
func resetWarnHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}

	target := resolveMentionOrReplyTarget(v)
	if target == "" {
		return ReplyErr(v, "❌ Mention or reply to the user to reset warnings!\nExample: .resetwarn @user")
	}

	targetJID, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid user.")
	}

	current := database.GetWarningCount(v.Info.Chat.String(), target)
	if current == 0 {
		return ReplyErr(v, fmt.Sprintf("✅ @%s has no warnings to reset.", targetJID.User))
	}

	if err := database.ClearWarnings(v.Info.Chat.String(), target); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	text := fmt.Sprintf("✅ *Warnings Reset*\n\n👤 User: @%s\n⚠️ Previous warnings: %d\n\nAll warnings cleared.",
		targetJID.User, current)

	wac := client.Get()
	_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: &text,
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: []string{target},
			},
		},
	})
	return err
}

// .hidetag / .tag <text>  — mention everyone without printing the full list.
// If replying to media (image/video), that media is resent with the caption
// and hidden mentions; if replying to text, that text is resent; otherwise
// the given args are sent as plain text.
func hidetagHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}

	wac := client.Get()
	info, err := wac.GetGroupInfo(context.Background(), v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to get group info: %v", err))
	}

	mentions := make([]string, 0, len(info.Participants))
	for _, p := range info.Participants {
		mentions = append(mentions, p.JID.String())
	}

	text := strings.Join(args, " ")

	quoted := GetQuotedMessage(v)
	if quoted != nil {
		// Media replies: re-send the media with mentions attached.
		if img := quoted.GetImageMessage(); img != nil {
			data, dlErr := wac.Download(context.Background(), img)
			if dlErr == nil {
				caption := text
				if caption == "" {
					caption = img.GetCaption()
				}
				uploaded, upErr := wac.Upload(context.Background(), data, whatsmeow.MediaImage)
				if upErr == nil {
					_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
						ImageMessage: &waProto.ImageMessage{
							Caption:       &caption,
							Mimetype:      img.Mimetype,
							URL:           &uploaded.URL,
							DirectPath:    &uploaded.DirectPath,
							MediaKey:      uploaded.MediaKey,
							FileEncSHA256: uploaded.FileEncSHA256,
							FileSHA256:    uploaded.FileSHA256,
							FileLength:    &uploaded.FileLength,
							ContextInfo:   &waProto.ContextInfo{MentionedJID: mentions},
						},
					})
					return err
				}
			}
		}
		if vid := quoted.GetVideoMessage(); vid != nil {
			data, dlErr := wac.Download(context.Background(), vid)
			if dlErr == nil {
				caption := text
				if caption == "" {
					caption = vid.GetCaption()
				}
				uploaded, upErr := wac.Upload(context.Background(), data, whatsmeow.MediaVideo)
				if upErr == nil {
					_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
						VideoMessage: &waProto.VideoMessage{
							Caption:       &caption,
							Mimetype:      vid.Mimetype,
							URL:           &uploaded.URL,
							DirectPath:    &uploaded.DirectPath,
							MediaKey:      uploaded.MediaKey,
							FileEncSHA256: uploaded.FileEncSHA256,
							FileSHA256:    uploaded.FileSHA256,
							FileLength:    &uploaded.FileLength,
							Seconds:       vid.Seconds,
							ContextInfo:   &waProto.ContextInfo{MentionedJID: mentions},
						},
					})
					return err
				}
			}
		}
		// Text reply: resend that text with mentions.
		if text == "" {
			if quoted.Conversation != nil {
				text = *quoted.Conversation
			} else if quoted.ExtendedTextMessage != nil && quoted.ExtendedTextMessage.Text != nil {
				text = *quoted.ExtendedTextMessage.Text
			}
		}
	}

	if text == "" {
		text = "​" // zero-width space, matches Knight's " " fallback so the message isn't empty
	}

	_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text:        &text,
			ContextInfo: &waProto.ContextInfo{MentionedJID: mentions},
		},
	})
	return err
}

func init() {
	Register(ModuleInfo{
		Name:        "Warnings",
		Description: "Warn system + silent hidetag",
		Commands: []CommandInfo{
			{Pattern: "warn", Handler: warnHandler, Sudo: true},
			{Pattern: "resetwarn", Handler: resetWarnHandler, Sudo: true},
			{Pattern: "hidetag", Handler: hidetagHandler, Sudo: true},
			{Pattern: "tag", Handler: hidetagHandler, Sudo: true},
		},
	})
}
