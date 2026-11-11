package modules

import (
	"context"
	"fmt"
	"strings"

	"PbxGb/client"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// .tagall [message]   — mention all group members
// .tagadmin [message] — mention only admins

func tagAllHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command only works in groups")
	}
	wac := client.Get()

	groupInfo, err := wac.GetGroupInfo(v.Info.Chat)
	if err != nil {
		return ReplyErr(v, "❌ Could not get group info: "+err.Error())
	}

	msg := "📢 *Tag All*"
	if len(args) > 0 {
		msg = strings.Join(args, " ")
	}

	var sb strings.Builder
	sb.WriteString(msg + "\n\n")

	var mentions []string
	for i, member := range groupInfo.Participants {
		jid := member.JID.ToNonAD()
		mentions = append(mentions, jid.String())
		sb.WriteString(fmt.Sprintf("@%s", jid.User))
		if i < len(groupInfo.Participants)-1 {
			sb.WriteString(" ")
		}
	}

	text := sb.String()
	_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: &text,
			ContextInfo: &waE2E.ContextInfo{
				MentionedJID: mentions,
			},
		},
	})
	return err
}

func tagAdminHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command only works in groups")
	}
	wac := client.Get()

	groupInfo, err := wac.GetGroupInfo(v.Info.Chat)
	if err != nil {
		return ReplyErr(v, "❌ Could not get group info: "+err.Error())
	}

	msg := "📢 *Admin Alert*"
	if len(args) > 0 {
		msg = strings.Join(args, " ")
	}

	var sb strings.Builder
	sb.WriteString(msg + "\n\n")

	var mentions []string
	var adminCount int
	for _, member := range groupInfo.Participants {
		if member.IsAdmin || member.IsSuperAdmin {
			jid := member.JID.ToNonAD()
			mentions = append(mentions, jid.String())
			sb.WriteString(fmt.Sprintf("@%s ", jid.User))
			adminCount++
		}
	}

	if adminCount == 0 {
		return ReplyErr(v, "❌ No admins found in this group")
	}

	text := sb.String()
	_, err = wac.SendMessage(context.Background(), v.Info.Chat, &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: &text,
			ContextInfo: &waE2E.ContextInfo{
				MentionedJID: mentions,
			},
		},
	})
	return err
}

// .listmembers — list all group members
func listMembersHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command only works in groups")
	}
	wac := client.Get()

	groupInfo, err := wac.GetGroupInfo(v.Info.Chat)
	if err != nil {
		return ReplyErr(v, "❌ Could not get group info: "+err.Error())
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("👥 *Members — %s* (%d)\n\n", groupInfo.Name, len(groupInfo.Participants)))

	admins := []types.GroupParticipant{}
	regular := []types.GroupParticipant{}
	for _, p := range groupInfo.Participants {
		if p.IsAdmin || p.IsSuperAdmin {
			admins = append(admins, p)
		} else {
			regular = append(regular, p)
		}
	}

	sb.WriteString("*👑 Admins:*\n")
	for _, p := range admins {
		icon := "🛡️"
		if p.IsSuperAdmin {
			icon = "👑"
		}
		sb.WriteString(fmt.Sprintf("%s +%s\n", icon, p.JID.User))
	}

	sb.WriteString(fmt.Sprintf("\n*👤 Members (%d):*\n", len(regular)))
	for i, p := range regular {
		sb.WriteString(fmt.Sprintf("%d. +%s\n", i+1, p.JID.User))
		if i >= 49 {
			sb.WriteString(fmt.Sprintf("... and %d more", len(regular)-50))
			break
		}
	}

	return ReplyErr(v, sb.String())
}

func init() {
	Register(ModuleInfo{
		Name:        "TagAll",
		Description: "Mention all or admin members in a group",
		Commands: []CommandInfo{
			{Pattern: "tagall", Handler: tagAllHandler, Sudo: true},
			{Pattern: "tagadmin", Handler: tagAdminHandler, Sudo: true},
			{Pattern: "listmembers", Handler: listMembersHandler, Sudo: true},
		},
	})
}
