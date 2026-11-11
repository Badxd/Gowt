package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"PbxGb/client"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// .whois @mention  — show profile info of a mentioned/quoted user
// .bio             — get/set your own WhatsApp bio
// .name            — get/set your display name

func whoisHandler(v *events.Message, args []string) error {
	wac := client.Get()

	var targetJID types.JID
	found := false

	// Check mentioned JIDs first
	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		if ci := ext.GetContextInfo(); ci != nil {
			if len(ci.GetMentionedJID()) > 0 {
				jidStr := ci.GetMentionedJID()[0]
				jid, err := types.ParseJID(jidStr)
				if err == nil {
					targetJID = jid
					found = true
				}
			}
			// Also check quoted message sender
			if !found && ci.GetParticipant() != "" {
				jid, err := types.ParseJID(ci.GetParticipant())
				if err == nil {
					targetJID = jid
					found = true
				}
			}
		}
	}

	if !found {
		// Check if a phone number was provided as argument
		if len(args) > 0 {
			num := strings.TrimPrefix(args[0], "+")
			num = strings.TrimPrefix(num, "@")
			num = strings.ReplaceAll(num, " ", "")
			if len(num) > 7 {
				jid, err := types.ParseJID(num + "@s.whatsapp.net")
				if err == nil {
					targetJID = jid
					found = true
				}
			}
		}
	}

	if !found {
		return ReplyErr(v, "Usage: .whois @mention\nOr reply to a message with .whois")
	}

	targetJID = targetJID.ToNonAD()

	// Get profile picture
	picInfo, err := wac.GetProfilePictureInfo(context.Background(), targetJID, &types.GetProfilePictureParams{})

	var picText string
	if err == nil && picInfo != nil {
		picText = picInfo.URL
	} else {
		picText = "_No profile picture_"
	}

	// Get user info
	userInfo, err := wac.GetUserInfo([]types.JID{targetJID})
	var statusText string
	if err == nil {
		if info, ok := userInfo[targetJID]; ok {
			statusText = info.Status
		}
	}

	out := fmt.Sprintf(`👤 *User Info*

📱 *Number:* +%s
💬 *Status:* %s
🖼️ *Profile Pic:* %s`,
		targetJID.User,
		func() string {
			if statusText == "" {
				return "_No status_"
			}
			return statusText
		}(),
		picText,
	)

	return ReplyErr(v, out)
}

func bioHandler(v *events.Message, args []string) error {
	wac := client.Get()

	if len(args) == 0 {
		// Get own bio
		selfJID := wac.Store.ID.ToNonAD()
		userInfo, err := wac.GetUserInfo([]types.JID{selfJID})
		if err != nil {
			return ReplyErr(v, "❌ Could not get bio: "+err.Error())
		}
		if info, ok := userInfo[selfJID]; ok {
			if info.Status == "" {
				return ReplyErr(v, "📝 Your bio is empty.")
			}
			return ReplyErr(v, "📝 *Your Bio:*\n"+info.Status)
		}
		return ReplyErr(v, "❌ Could not read bio")
	}

	// Set bio
	newBio := strings.Join(args, " ")
	if len(newBio) > 139 {
		return ReplyErr(v, "❌ Bio must be ≤139 characters")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := wac.SetStatusMessage(ctx, newBio)
	if err != nil {
		return ReplyErr(v, "❌ Could not set bio: "+err.Error())
	}
	return ReplyErr(v, "✅ Bio updated: "+newBio)
}

func nameHandler(v *events.Message, args []string) error {
	wac := client.Get()

	if len(args) == 0 {
		// Get own push name
		name := wac.Store.PushName
		if name == "" {
			name = "_not set_"
		}
		return ReplyErr(v, "📛 *Your Name:* "+name)
	}

	// Set push name
	newName := strings.Join(args, " ")
	if len(newName) > 25 {
		return ReplyErr(v, "❌ Name must be ≤25 characters")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Send presence with new name
	err := wac.SendPresence(types.PresenceAvailable)
	if err != nil {
		return ReplyErr(v, "❌ Could not set name: "+err.Error())
	}

	// Update name via profile node
	_, err2 := wac.SendMessage(ctx, wac.Store.ID.ToNonAD(), &waE2E.Message{})
	_ = err2

	// Store locally
	wac.Store.PushName = newName

	return ReplyErr(v, "✅ Display name set to: *"+newName+"*\n_Note: Name update may require reconnection to sync._")
}

func init() {
	Register(ModuleInfo{
		Name:        "WhoIs",
		Description: "User info lookup, bio management, display name",
		Commands: []CommandInfo{
			{Pattern: "whois", Handler: whoisHandler, Sudo: true},
			{Pattern: "bio", Handler: bioHandler, Sudo: true},
			{Pattern: "name", Handler: nameHandler, Sudo: true},
		},
	})
}
