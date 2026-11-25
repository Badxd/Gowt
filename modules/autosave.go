package modules

// PBX Elite — AutoSave Module
//
// .save         — save replied media to Saved Messages
// .autosave on  — auto-save ALL incoming view-once to Saved Messages
// .autosave off — disable autosave

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"PbxGb/client"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	autoSaveMu      sync.RWMutex
	autoSaveEnabled = false
)

// getSelfJID returns the bot's own JID for Saved Messages
func getSelfJID() types.JID {
	wac := client.Get()
	if wac == nil || wac.Store.ID == nil {
		return types.JID{}
	}
	return types.NewJID(wac.Store.ID.User, types.DefaultUserServer)
}

// HandleAutoSave — saves view-once images/videos when autosave is enabled
func HandleAutoSave(v *events.Message) {
	autoSaveMu.RLock()
	enabled := autoSaveEnabled
	autoSaveMu.RUnlock()

	if !enabled {
		return
	}

	wac := client.Get()
	if wac == nil || wac.Store.ID == nil {
		return
	}

	// Don't save own messages
	if v.Info.IsFromMe {
		return
	}

	selfJID := getSelfJID()

	// VIEW-ONCE IMAGE
	if img := v.Message.GetImageMessage(); img != nil && img.GetViewOnce() {
		data, err := wac.Download(context.Background(), img)
		if err != nil {
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaImage)
		if err != nil {
			return
		}
		caption := fmt.Sprintf("💾 *AutoSaved View-Once Image*\n📩 From: *%s*\n💬 %s",
			v.Info.PushName, v.Info.Chat.String())
		_, _ = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				Caption:       &caption,
				Mimetype:      img.Mimetype,
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
			},
		})
		return
	}

	// VIEW-ONCE VIDEO
	if vid := v.Message.GetVideoMessage(); vid != nil && vid.GetViewOnce() {
		data, err := wac.Download(context.Background(), vid)
		if err != nil {
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaVideo)
		if err != nil {
			return
		}
		caption := fmt.Sprintf("💾 *AutoSaved View-Once Video*\n📩 From: *%s*\n💬 %s",
			v.Info.PushName, v.Info.Chat.String())
		_, _ = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
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
			},
		})
	}
}

// saveHandler — .save: save ANY replied media to Saved Messages
func saveHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	// Get the quoted message
	qm := GetQuotedMessage(v)
	if qm == nil {
		return ReplyErr(v, "❌ Reply to an image or video to save it.")
	}

	selfJID := getSelfJID()
	if selfJID.IsEmpty() {
		return ReplyErr(v, "❌ Could not determine your Saved Messages JID.")
	}

	ctx := context.Background()

	// IMAGE
	if img := qm.GetImageMessage(); img != nil {
		data, err := wac.Download(ctx, img)
		if err != nil {
			return ReplyErr(v, "❌ Could not download image: "+err.Error())
		}
		uploaded, err := wac.Upload(ctx, data, whatsmeow.MediaImage)
		if err != nil {
			return ReplyErr(v, "❌ Upload failed: "+err.Error())
		}
		caption := "💾 *Saved from chat*"
		_, err = wac.SendMessage(ctx, selfJID, &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				Caption:       &caption,
				Mimetype:      img.Mimetype,
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
			},
		})
		if err != nil {
			return ReplyErr(v, "❌ Failed to save: "+err.Error())
		}
		return ReplyErr(v, "✅ Image saved to your Saved Messages.")
	}

	// VIDEO
	if vid := qm.GetVideoMessage(); vid != nil {
		data, err := wac.Download(ctx, vid)
		if err != nil {
			return ReplyErr(v, "❌ Could not download video: "+err.Error())
		}
		uploaded, err := wac.Upload(ctx, data, whatsmeow.MediaVideo)
		if err != nil {
			return ReplyErr(v, "❌ Upload failed: "+err.Error())
		}
		caption := "💾 *Saved from chat*"
		_, err = wac.SendMessage(ctx, selfJID, &waProto.Message{
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
			},
		})
		if err != nil {
			return ReplyErr(v, "❌ Failed to save: "+err.Error())
		}
		return ReplyErr(v, "✅ Video saved to your Saved Messages.")
	}

	// AUDIO
	if aud := qm.GetAudioMessage(); aud != nil {
		data, err := wac.Download(ctx, aud)
		if err != nil {
			return ReplyErr(v, "❌ Could not download audio: "+err.Error())
		}
		uploaded, err := wac.Upload(ctx, data, whatsmeow.MediaAudio)
		if err != nil {
			return ReplyErr(v, "❌ Upload failed: "+err.Error())
		}
		_, err = wac.SendMessage(ctx, selfJID, &waProto.Message{
			AudioMessage: &waProto.AudioMessage{
				Mimetype:      aud.Mimetype,
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
				Seconds:       aud.Seconds,
			},
		})
		if err != nil {
			return ReplyErr(v, "❌ Failed to save: "+err.Error())
		}
		return ReplyErr(v, "✅ Audio saved to your Saved Messages.")
	}

	// DOCUMENT
	if doc := qm.GetDocumentMessage(); doc != nil {
		data, err := wac.Download(ctx, doc)
		if err != nil {
			return ReplyErr(v, "❌ Could not download document: "+err.Error())
		}
		uploaded, err := wac.Upload(ctx, data, whatsmeow.MediaDocument)
		if err != nil {
			return ReplyErr(v, "❌ Upload failed: "+err.Error())
		}
		title := doc.GetTitle()
		_, err = wac.SendMessage(ctx, selfJID, &waProto.Message{
			DocumentMessage: &waProto.DocumentMessage{
				Mimetype:      doc.Mimetype,
				Title:         &title,
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
			},
		})
		if err != nil {
			return ReplyErr(v, "❌ Failed to save: "+err.Error())
		}
		return ReplyErr(v, "✅ Document saved to your Saved Messages.")
	}

	// TEXT
	text := ""
	if qm.GetConversation() != "" {
		text = qm.GetConversation()
	} else if qm.GetExtendedTextMessage() != nil && qm.GetExtendedTextMessage().GetText() != "" {
		text = qm.GetExtendedTextMessage().GetText()
	}
	if text != "" {
		note := "💾 *Saved Note:*\n\n" + text
		_, err := wac.SendMessage(ctx, selfJID, &waProto.Message{
			Conversation: &note,
		})
		if err != nil {
			return ReplyErr(v, "❌ Failed to save text: "+err.Error())
		}
		return ReplyErr(v, "✅ Text saved to your Saved Messages.")
	}

	return ReplyErr(v, "❌ Could not find media or text in quoted message.")
}

func autoSaveToggleHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		autoSaveMu.RLock()
		s := autoSaveEnabled
		autoSaveMu.RUnlock()
		state := "🔴 OFF"
		if s {
			state = "🟢 ON"
		}
		return ReplyErr(v, fmt.Sprintf(
			"💾 *AutoSave*: %s\n\n"+
				"When ON, all view-once images/videos are automatically saved to your Saved Messages.\n"+
				"Use `.save` to manually save any replied media.",
			state,
		))
	}

	switch strings.ToLower(args[0]) {
	case "on":
		autoSaveMu.Lock()
		autoSaveEnabled = true
		autoSaveMu.Unlock()
		return ReplyErr(v, "✅ *AutoSave ON*\nView-once media will be auto-saved to Saved Messages.")
	case "off":
		autoSaveMu.Lock()
		autoSaveEnabled = false
		autoSaveMu.Unlock()
		return ReplyErr(v, "🔴 *AutoSave OFF*")
	}
	return ReplyErr(v, "❌ Use: .autosave on / off")
}

func init() {
	Register(ModuleInfo{
		Name:        "AutoSave",
		Description: "Save replied media to Saved Messages (.save) or auto-save view-once (.autosave)",
		Commands: []CommandInfo{
			{Pattern: "save", Handler: saveHandler, Sudo: true},
			{Pattern: "autosave", Handler: autoSaveToggleHandler, Sudo: true},
		},
	})
}
