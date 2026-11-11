package modules

// Status Tools — post, view, delete WhatsApp status (story) updates.
//
// Commands:
//   .setstatus <text>       — post a text status update
//   .statusimg              — post image status (reply to image)
//   .statusvid              — post video status (reply to video)
//   .mystatus               — show your current status message (bio)

import (
	"context"
	"fmt"
	"strings"
	"time"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"PbxGb/client"
)

// ── TEXT STATUS ───────────────────────────────────────────────────────────────
// Posts to WhatsApp Stories / Status broadcast
func setStatusHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .setstatus <text>")
	}
	text := strings.Join(args, " ")
	wac := client.Get()
	ctx := context.Background()

	// WhatsApp status is sent to the special StatusBroadcastJID
	_, err := wac.SendMessage(ctx, types.StatusBroadcastJID, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(text),
			FontType: waProto.ExtendedTextMessage_SANS_SERIF.Enum(),
			BackgroundArgb: proto.Uint32(0xFF25D366), // WhatsApp green
		},
	})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to post status: %v", err))
	}
	return ReplyErr(v, "✅ Status posted to WhatsApp Stories.")
}

// ── IMAGE STATUS ──────────────────────────────────────────────────────────────
func statusImgHandler(v *events.Message, args []string) error {
	wac := client.Get()
	ctx := context.Background()

	// Get image from reply
	var imgMsg *waProto.ImageMessage
	if v.Message != nil {
		if img := v.Message.GetImageMessage(); img != nil {
			imgMsg = img
		} else if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			ci := ext.GetContextInfo()
			if ci != nil && ci.QuotedMessage != nil {
				imgMsg = ci.QuotedMessage.GetImageMessage()
			}
		}
	}
	if imgMsg == nil {
		return ReplyErr(v, "❌ Reply to an image to post it as a status.")
	}

	caption := ""
	if len(args) > 0 {
		caption = strings.Join(args, " ")
	}

	// Upload to WhatsApp and re-send to StatusBroadcastJID
	// We re-use existing image message data (URL is still valid)
	newMsg := proto.Clone(imgMsg).(*waProto.ImageMessage)
	if caption != "" {
		newMsg.Caption = proto.String(caption)
	}

	_, err := wac.SendMessage(ctx, types.StatusBroadcastJID, &waProto.Message{
		ImageMessage: newMsg,
	})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, "✅ Image status posted.")
}

// ── VIDEO STATUS ──────────────────────────────────────────────────────────────
func statusVidHandler(v *events.Message, args []string) error {
	wac := client.Get()
	ctx := context.Background()

	var vidMsg *waProto.VideoMessage
	if v.Message != nil {
		if vid := v.Message.GetVideoMessage(); vid != nil {
			vidMsg = vid
		} else if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			ci := ext.GetContextInfo()
			if ci != nil && ci.QuotedMessage != nil {
				vidMsg = ci.QuotedMessage.GetVideoMessage()
			}
		}
	}
	if vidMsg == nil {
		return ReplyErr(v, "❌ Reply to a video to post it as a status.")
	}

	caption := ""
	if len(args) > 0 {
		caption = strings.Join(args, " ")
	}

	newMsg := proto.Clone(vidMsg).(*waProto.VideoMessage)
	if caption != "" {
		newMsg.Caption = proto.String(caption)
	}

	_, err := wac.SendMessage(ctx, types.StatusBroadcastJID, &waProto.Message{
		VideoMessage: newMsg,
	})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, "✅ Video status posted.")
}

// ── MY STATUS ─────────────────────────────────────────────────────────────────
func myStatusHandler(v *events.Message, args []string) error {
	wac := client.Get()
	if wac.Store.ID == nil {
		return ReplyErr(v, "❌ Not connected.")
	}
	ctx := context.Background()

	// GetUserInfo for ourselves
	contacts, err := wac.GetUserInfo(ctx, []types.JID{*wac.Store.ID})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	info, ok := contacts[*wac.Store.ID]
	if !ok {
		return ReplyErr(v, "ℹ️ No self-info found.")
	}

	pushName := wac.Store.PushName
	verName := "—"
	if info.VerifiedName != nil {
		verName = info.VerifiedName.Name
	}

	contact, _ := wac.Store.Contacts.GetContact(*wac.Store.ID)
	bio := contact.BusinessName
	if bio == "" {
		bio = "—"
	}

	return ReplyErr(v, fmt.Sprintf(
		"👤 *My Profile*\n\n"+
			"📱 *Number:* +%s\n"+
			"🏷️ *Push name:* %s\n"+
			"✅ *Verified name:* %s\n"+
			"📝 *Bio:* %s\n"+
			"⏰ *Checked:* %s",
		wac.Store.ID.User,
		pushName,
		verName,
		bio,
		time.Now().Format("02 Jan 3:04 PM"),
	))
}

func init() {
	Register(ModuleInfo{
		Name:        "StatusTools",
		Description: "Post text/image/video WhatsApp status (stories), view own profile",
		Commands: []CommandInfo{
			{Pattern: "setstatus",  Handler: setStatusHandler, Sudo: true},
			{Pattern: "statusimg",  Handler: statusImgHandler, Sudo: true},
			{Pattern: "statusvid",  Handler: statusVidHandler, Sudo: true},
			{Pattern: "mystatus",   Handler: myStatusHandler,  Sudo: true},
			{Pattern: "myprofile",  Handler: myStatusHandler,  Sudo: true},
		},
	})
}
