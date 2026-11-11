package modules

// Profile Tools — set/get WhatsApp profile using whatsmeow APIs.
//
// Commands:
//   .setbio <text>        — change your about/bio text
//   .setmypp              — set profile picture (reply to image)
//   .delmypp              — remove your profile picture
//   .getpp @user          — get a user's profile picture URL
//   .getname @user        — get a user's display name from store
//   .getbio @user         — get a user's about text
//   .copyprofile @user    — show another user's name + profile pic URL

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

// ── SET BIO ───────────────────────────────────────────────────────────────────
func setMyBioHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .setbio <text>")
	}
	text := strings.Join(args, " ")
	wac := client.Get()
	ctx := context.Background()

	if err := wac.SetStatusMessage(ctx, text); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("✅ Bio/status set to: _%s_", text))
}

// ── SET PROFILE PICTURE ───────────────────────────────────────────────────────
func setMyPPHandler(v *events.Message, args []string) error {
	img := getQuotedImageBytes(v)
	if img == nil {
		return ReplyErr(v, "❌ Reply to an image to set it as your profile picture.")
	}
	wac := client.Get()
	ctx := context.Background()
	_, err := wac.SetGroupPhoto(ctx, *wac.Store.ID, img)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, "✅ Profile picture updated.")
}

// ── DELETE PROFILE PICTURE ────────────────────────────────────────────────────
func delMyPPHandler(v *events.Message, args []string) error {
	wac := client.Get()
	ctx := context.Background()
	_, err := wac.SetGroupPhoto(ctx, *wac.Store.ID, nil)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, "✅ Profile picture removed.")
}

// ── GET PROFILE PICTURE ───────────────────────────────────────────────────────
func getPPHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .getpp @mention or .getpp 919xxx")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}
	wac := client.Get()
	ctx := context.Background()
	ppInfo, err := wac.GetProfilePictureInfo(ctx, jid, &types.GetProfilePictureParams{IsCommunity: false})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	if ppInfo == nil || ppInfo.URL == "" {
		return ReplyErr(v, fmt.Sprintf("ℹ️ *+%s* has no profile picture.", jid.User))
	}
	return ReplyErr(v, fmt.Sprintf(
		"🖼️ *Profile pic of +%s*\n\nURL: %s\nID: `%s`",
		jid.User, ppInfo.URL, ppInfo.ID,
	))
}

// ── GET NAME ──────────────────────────────────────────────────────────────────
func getNameHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .getname @mention or .getname 919xxx")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}

	wac := client.Get()
	// Try store first
	contact, storeErr := wac.Store.Contacts.GetContact(jid)
	if storeErr == nil && contact.FullName != "" {
		return ReplyErr(v, fmt.Sprintf("👤 *+%s*\nName: *%s*", jid.User, contact.FullName))
	}

	// Fall back to GetUserInfo
	ctx := context.Background()
	contacts, err := wac.GetUserInfo(ctx, []types.JID{jid})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	info, ok := contacts[jid]
	if !ok || info.VerifiedName == nil {
		return ReplyErr(v, fmt.Sprintf("ℹ️ No name info for *+%s*.", jid.User))
	}
	return ReplyErr(v, fmt.Sprintf("👤 *+%s*\nVerified name: *%s*", jid.User, info.VerifiedName.Name))
}

// ── GET BIO ───────────────────────────────────────────────────────────────────
func getBioHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .getbio @mention or .getbio 919xxx")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}
	wac := client.Get()
	contact, err := wac.Store.Contacts.GetContact(jid)
	if err == nil && contact.BusinessName != "" {
		return ReplyErr(v, fmt.Sprintf("📝 *Bio of +%s:*\n\n%s", jid.User, contact.BusinessName))
	}
	return ReplyErr(v, fmt.Sprintf("ℹ️ No bio info available for *+%s*.", jid.User))
}

// ── COPY PROFILE (show info) ──────────────────────────────────────────────────
func copyProfileHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .copyprofile @mention")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}
	wac := client.Get()
	ctx := context.Background()

	// Get their name from store
	displayName := "+" + jid.User
	contact, _ := wac.Store.Contacts.GetContact(jid)
	if contact.FullName != "" {
		displayName = contact.FullName
	}

	// Get their pp URL
	ppMsg := "_(no profile picture)_"
	ppInfo, ppErr := wac.GetProfilePictureInfo(ctx, jid, &types.GetProfilePictureParams{IsCommunity: false})
	if ppErr == nil && ppInfo != nil && ppInfo.URL != "" {
		ppMsg = "📸 PP URL: " + ppInfo.URL
	}

	return ReplyErr(v, fmt.Sprintf(
		"📋 *Profile of +%s*\nName: *%s*\n%s",
		jid.User, displayName, ppMsg,
	))
}

// ── SEND VCARD (alias here, full impl in msgtools.go) ────────────────────────
// sendVCardHandler re-exported alias — keep command registration here
func sendVCardHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .vcard @mention or .vcard 919xxx")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid JID.")
	}
	wac := client.Get()
	ctx := context.Background()

	displayName := "+" + jid.User
	contact, _ := wac.Store.Contacts.GetContact(jid)
	if contact.FullName != "" {
		displayName = contact.FullName
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
	return ReplyErr(v, fmt.Sprintf("📇 vCard for *%s* sent.", displayName))
}

// helper: get thumbnail bytes from a quoted or current image message
func getQuotedImageBytes(v *events.Message) []byte {
	if v.Message == nil {
		return nil
	}
	// direct image
	if img := v.Message.GetImageMessage(); img != nil {
		if t := img.GetJpegThumbnail(); len(t) > 0 {
			return t
		}
	}
	// quoted image
	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		ctx := ext.GetContextInfo()
		if ctx != nil && ctx.QuotedMessage != nil {
			if img := ctx.QuotedMessage.GetImageMessage(); img != nil {
				if t := img.GetJpegThumbnail(); len(t) > 0 {
					return t
				}
			}
		}
	}
	return nil
}

func init() {
	Register(ModuleInfo{
		Name:        "ProfileTools",
		Description: "Set/get bio, profile picture — your own and others",
		Commands: []CommandInfo{
			{Pattern: "setbio",      Handler: setMyBioHandler,  Sudo: true},
			{Pattern: "setmypp",     Handler: setMyPPHandler,   Sudo: true},
			{Pattern: "delmypp",     Handler: delMyPPHandler,   Sudo: true},
			{Pattern: "getpp",       Handler: getPPHandler,     Sudo: true},
			{Pattern: "getname",     Handler: getNameHandler,   Sudo: true},
			{Pattern: "getbio",      Handler: getBioHandler,    Sudo: true},
			{Pattern: "copyprofile", Handler: copyProfileHandler, Sudo: true},
			{Pattern: "vcard",       Handler: sendVCardHandler, Sudo: true},
		},
	})
}
