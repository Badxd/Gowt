package modules

// PBX Elite — View Once Module
//
// .vv      — reveal a quoted view-once image/video
// .antiviewonce on/off — auto-save all view-once to saved messages

import (
	"context"
	"fmt"
	"sync"

	"PbxGb/client"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

var (
	antiVOMu      sync.RWMutex
	antiVOEnabled = false
)

// HandleAutoVV — saves all view-once messages automatically (triggered in HandleMessage)
func HandleAutoVV(v *events.Message) {
	antiVOMu.RLock()
	enabled := antiVOEnabled
	antiVOMu.RUnlock()

	if !enabled {
		return
	}

	wac := client.Get()
	if wac == nil || wac.Store.ID == nil {
		return
	}

	// Don't process own messages
	if v.Info.IsFromMe {
		return
	}

	selfJID := getSelfJID()
	if selfJID.IsEmpty() {
		return
	}

	// helper: send error notice to saved messages
	notifyFail := func(kind, reason string) {
		note := fmt.Sprintf("⚠️ *Anti-ViewOnce: %s capture failed*\n📩 From: *%s*\n💬 Chat: %s\n❌ %s",
			kind, v.Info.PushName, v.Info.Chat.String(), reason)
		_, _ = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
			Conversation: proto.String(note),
		})
	}

	img := v.Message.GetImageMessage()
	if img != nil && img.GetViewOnce() {
		data, err := wac.Download(context.Background(), img)
		if err != nil {
			notifyFail("Image", err.Error())
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaImage)
		if err != nil {
			notifyFail("Image", "upload failed: "+err.Error())
			return
		}
		caption := fmt.Sprintf("👁️ *Anti-ViewOnce: Image*\n📩 From: *%s*\n💬 Chat: %s",
			v.Info.PushName, v.Info.Chat.String())
		_, _ = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				Caption:       proto.String(caption),
				Mimetype:      img.Mimetype,
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
				ViewOnce:      proto.Bool(false),
			},
		})
		return
	}

	vid := v.Message.GetVideoMessage()
	if vid != nil && vid.GetViewOnce() {
		data, err := wac.Download(context.Background(), vid)
		if err != nil {
			notifyFail("Video", err.Error())
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaVideo)
		if err != nil {
			notifyFail("Video", "upload failed: "+err.Error())
			return
		}
		caption := fmt.Sprintf("👁️ *Anti-ViewOnce: Video*\n📩 From: *%s*\n💬 Chat: %s",
			v.Info.PushName, v.Info.Chat.String())
		_, _ = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
			VideoMessage: &waProto.VideoMessage{
				Caption:       proto.String(caption),
				Mimetype:      vid.Mimetype,
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
				Seconds:       vid.Seconds,
				ViewOnce:      proto.Bool(false),
			},
		})
	}
}

// HandleEmojiVV is kept for compatibility
func HandleEmojiVV(v *events.Message) {
	HandleAutoVV(v)
}

// HandleAntiViewOnce is kept for compatibility
func HandleAntiViewOnce(v *events.Message) {
	HandleAutoVV(v)
}

// vvHandler — .vv: reveal quoted view-once by fetching from message store
func vvHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	// Get quoted context info
	ext := v.Message.GetExtendedTextMessage()
	if ext == nil {
		return ReplyErr(v, "❌ Reply to a view-once message.")
	}
	ci := ext.GetContextInfo()
	if ci == nil || ci.GetQuotedMessage() == nil {
		return ReplyErr(v, "❌ Reply to a view-once message.")
	}

	qm := ci.GetQuotedMessage()

	// Try image first
	if img := qm.GetImageMessage(); img != nil {
		// WhatsApp strips the media URL from quoted view-once messages.
		// Download only works if the message is very recent and URL hasn't expired.
		data, err := wac.Download(context.Background(), img)
		if err != nil {
			return ReplyErr(v, "❌ *Cannot reveal this view-once.*\n\n"+
				"WhatsApp removes the media URL from view-once messages after they're opened.\n\n"+
				"💡 Enable `.antiviewonce on` — the bot will silently save all future view-once media *before* you open it.")
		}

		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaImage)
		if err != nil {
			return ReplyErr(v, "❌ Upload failed: "+err.Error())
		}

		caption := fmt.Sprintf("👁️ *Revealed View Once* from *%s*", ci.GetParticipant())
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
				ViewOnce:      proto.Bool(false),
			},
		})
		return err
	}

	// Try video
	if vid := qm.GetVideoMessage(); vid != nil {
		data, err := wac.Download(context.Background(), vid)
		if err != nil {
			return ReplyErr(v, "❌ *Cannot reveal this view-once.*\n\n"+
				"WhatsApp removes the media URL from view-once messages after they're opened.\n\n"+
				"💡 Enable `.antiviewonce on` — the bot will silently save all future view-once media *before* you open it.")
		}

		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaVideo)
		if err != nil {
			return ReplyErr(v, "❌ Upload failed: "+err.Error())
		}

		caption := fmt.Sprintf("👁️ *Revealed View Once* from *%s*", ci.GetParticipant())
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
				ViewOnce:      proto.Bool(false),
			},
		})
		return err
	}

	return ReplyErr(v, "❌ Quoted message is not a view-once image or video.\n💡 Enable `.antiviewonce on` to auto-capture future view-once media.")
}

// antiviewonce toggle
func antiVOToggleHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		antiVOMu.RLock()
		s := antiVOEnabled
		antiVOMu.RUnlock()
		state := "🔴 OFF"
		if s {
			state = "🟢 ON"
		}
		return ReplyErr(v, fmt.Sprintf("👁️ Anti View Once: *%s*\n\nView-once media is silently saved to your Saved Messages.", state))
	}

	switch args[0] {
	case "on":
		antiVOMu.Lock()
		antiVOEnabled = true
		antiVOMu.Unlock()
		return ReplyErr(v, "✅ *Anti View Once ON*\n\nAll incoming view-once images/videos will be\nauto-saved to your *Saved Messages*.\nSender will NOT know.")
	case "off":
		antiVOMu.Lock()
		antiVOEnabled = false
		antiVOMu.Unlock()
		return ReplyErr(v, "🔴 *Anti View Once OFF*")
	}
	return ReplyErr(v, "❌ Use: .antiviewonce on / off")
}

func init() {
	Register(ModuleInfo{
		Name:        "ViewOnce",
		Description: "Reveal view-once media (.vv) and auto-save them (.antiviewonce)",
		Commands: []CommandInfo{
			{Pattern: "vv", Handler: vvHandler, Sudo: true},
			{Pattern: "antiviewonce", Handler: antiVOToggleHandler, Sudo: true},
		},
	})
}
