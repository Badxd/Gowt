package modules

import (
	"context"
	"fmt"
	"strings"

	"PbxGb/client"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// .setname <name>
func setNameHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .setname <name>")
	}
	name := strings.Join(args, " ")
	wac := client.Get()
	err := wac.SetGroupName(context.Background(), v.Info.Chat, name)
	if err != nil {
		// try profile name
		return ReplyErr(v, "❌ Could not set name: "+err.Error())
	}
	return ReplyErr(v, fmt.Sprintf("✅ Name set to: *%s*", name))
}

// .setbio <bio>
func setBioHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .setbio <text>")
	}
	bio := strings.Join(args, " ")
	wac := client.Get()
	err := wac.SetStatusMessage(context.Background(), bio)
	if err != nil {
		return ReplyErr(v, "❌ Failed to set bio: "+err.Error())
	}
	return ReplyErr(v, fmt.Sprintf("✅ Bio updated:\n_%s_", bio))
}

// .getpfp [@user or reply]
func getPfpHandler(v *events.Message, args []string) error {
	wac := client.Get()
	var target types.JID

	if IsReply(v) {
		q := GetQuotedMessage(v)
		if q != nil && q.ExtendedTextMessage != nil && q.ExtendedTextMessage.ContextInfo != nil {
			raw := q.ExtendedTextMessage.ContextInfo.GetParticipant()
			if raw != "" {
				jid, err := types.ParseJID(raw)
				if err == nil {
					target = jid
				}
			}
		}
	} else if len(args) > 0 {
		raw := strings.TrimPrefix(args[0], "@")
		raw = raw + "@s.whatsapp.net"
		jid, err := types.ParseJID(raw)
		if err != nil {
			return ReplyErr(v, "❌ Invalid JID: "+args[0])
		}
		target = jid
	} else {
		target = *wac.Store.ID
	}

	pic, err := wac.GetProfilePictureInfo(context.Background(), target, &whatsmeow.GetProfilePictureParams{})
	if err != nil || pic == nil {
		return ReplyErr(v, "❌ No profile picture found.")
	}

	return ReplyErr(v, fmt.Sprintf("🖼️ Profile picture URL:\n%s", pic.URL))
}

// .clonebio @user — copy bio from another user (show current status)
func cloneBioHandler(v *events.Message, args []string) error {
	return ReplyErr(v, "ℹ️ WhatsApp API doesn't expose others' bio text.\nYou can only set your own bio with .setbio <text>")
}

// .clonename — copy group name from another (for admin use)
func cloneNameHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .clonename <name to copy>")
	}
	name := strings.Join(args, " ")
	wac := client.Get()
	err := wac.SetGroupName(context.Background(), v.Info.Chat, name)
	if err != nil {
		return ReplyErr(v, "❌ "+err.Error())
	}
	return ReplyErr(v, fmt.Sprintf("✅ Group name set to: *%s*", name))
}

func init() {
	Register(ModuleInfo{
		Name:        "Profile",
		Description: "Profile management: name, bio, pfp",
		Commands: []CommandInfo{
			{Pattern: "setbio", Handler: setBioHandler, Sudo: true},
			{Pattern: "getpfp", Handler: getPfpHandler, Sudo: true},
			{Pattern: "clonebio", Handler: cloneBioHandler, Sudo: true},
			{Pattern: "clonename", Handler: cloneNameHandler, Sudo: true},
			{Pattern: "setgname", Handler: setNameHandler, Sudo: true},
		},
	})
}
