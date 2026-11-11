package modules

import (
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// .getpp [@user | number]   — get profile picture of a user
// .pfp  [@user | number]    — alias

func getppHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := newContext(20)
	defer cancel()

	var targetJID types.JID

	// check mentioned user
	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		if ci := ext.GetContextInfo(); ci != nil {
			mentions := ci.GetMentionedJID()
			if len(mentions) > 0 {
				j, err := types.ParseJID(mentions[0])
				if err == nil {
					targetJID = j
				}
			}
		}
	}

	// check args (phone number)
	if targetJID.IsEmpty() && len(args) > 0 {
		num := strings.TrimPrefix(args[0], "+")
		num = strings.TrimPrefix(num, "@")
		j, err := types.ParseJID(num + "@s.whatsapp.net")
		if err == nil {
			targetJID = j
		}
	}

	// self
	if targetJID.IsEmpty() {
		if wac.Store.ID != nil {
			targetJID = *wac.Store.ID
		} else {
			return ReplyErr(v, "❌ Cannot determine own JID.")
		}
	}

	pic, err := wac.GetProfilePictureInfo(ctx, targetJID, &whatsmeow.GetProfilePictureParams{Preview: false})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Could not get profile picture: %v", err))
	}
	if pic == nil || pic.URL == "" {
		return ReplyErr(v, "❌ No profile picture found (user may have privacy enabled).")
	}

	// Send the URL as a link — user can open it
	return ReplyErr(v, fmt.Sprintf("🖼️ Profile Picture:\n%s", pic.URL))
}

func init() {
	Register(ModuleInfo{
		Name:        "Get Profile Picture",
		Description: "Fetch profile picture URL of a user",
		Commands: []CommandInfo{
			{Pattern: "getpp", Handler: getppHandler, Sudo: true},
			{Pattern: "pfp", Handler: getppHandler, Sudo: true},
		},
	})
}
