package modules

import (
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// profilepic.go — set and remove the bot's own profile picture.
//
// .setpp   — reply to an image to set it as the bot's profile picture
// .delpp   — remove the bot's profile picture
//
// Note: .getpp / .pfp are handled by getpp.go

func setppHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	// Get image from quoted message
	ctxInfo := v.Message.GetExtendedTextMessage().GetContextInfo()
	if ctxInfo == nil {
		return ReplyErr(v, "❌ Reply to an image to set it as profile picture.")
	}

	quotedMsg := ctxInfo.GetQuotedMessage()
	if quotedMsg == nil {
		return ReplyErr(v, "❌ Reply to an image to set it as profile picture.")
	}

	imgMsg := quotedMsg.GetImageMessage()
	if imgMsg == nil {
		return ReplyErr(v, "❌ Quoted message is not an image.")
	}

	data, err := wac.Download(imgMsg)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to download image: %v", err))
	}

	if wac.Store.ID == nil {
		return ReplyErr(v, "❌ Bot not logged in.")
	}

	ctx, cancel := newContext(30)
	defer cancel()

	resp, err := wac.SetGroupPhoto(ctx, *wac.Store.ID, data)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to set profile picture: %v", err))
	}

	return ReplyErr(v, fmt.Sprintf("✅ Profile picture updated! (ID: %s)", resp))
}

func delppHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	if wac.Store.ID == nil {
		return ReplyErr(v, "❌ Bot not logged in.")
	}

	ctx, cancel := newContext(15)
	defer cancel()

	// Pass nil data to remove profile picture
	_, err := wac.SetGroupPhoto(ctx, *wac.Store.ID, nil)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to remove profile picture: %v", err))
	}

	return ReplyErr(v, "✅ Profile picture removed.")
}

// SetGroupPhoto signature: func (wac *Client) SetGroupPhoto(ctx context.Context, jid types.JID, avatar []byte) (string, error)
// Passing nil avatar removes the picture. This same API works for own JID.
var _ = (*whatsmeow.Client)(nil) // keep whatsmeow import satisfied if SetGroupPhoto moves

func init() {
	Register(ModuleInfo{
		Name:        "ProfilePic",
		Description: "Set or delete the bot's profile picture (get is in getpp.go)",
		Commands: []CommandInfo{
			{Pattern: "setpp", Handler: setppHandler, Sudo: true},
			{Pattern: "delpp", Handler: delppHandler, Sudo: true},
		},
	})
}
