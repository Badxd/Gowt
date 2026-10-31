package modules

// PBX Elite — Auto Save to Self Chat
// ✅ .autosave on: automatically saves VIEW-ONCE media only (not every
//    status/group photo — that was the old, too-broad behavior)
// ✅ .save command: manually save ANY replied media (view-once or not)

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"PbxGb/client"
	"PbxGb/config"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	autoSaveMu      sync.RWMutex
	autoSaveEnabled = false
)

// getSelfJID — returns owner's self-chat JID
func getSelfJID() types.JID {
	ownerJIDStr := config.C.OwnerJID
	ownerUser := ownerJIDStr
	for i, c := range ownerJIDStr {
		if c == '@' {
			ownerUser = ownerJIDStr[:i]
			break
		}
	}
	return types.NewJID(ownerUser, types.DefaultUserServer)
}

// HandleAutoSave — called from HandleMessage for every incoming message
func HandleAutoSave(v *events.Message) {

	autoSaveMu.RLock()
	enabled := autoSaveEnabled
	autoSaveMu.RUnlock()

	if !enabled {
		return
	}

	if v.Message == nil {
		return
	}

	wac := client.Get()

	// Don't save own messages
	if wac.Store.ID != nil &&
		v.Info.Sender.User == wac.Store.ID.User {
		return
	}

	selfJID := getSelfJID()

	saveMediaToSelf(v, wac, selfJID, false)
}

// saveMediaToSelf — core logic, reused by .save command too
func saveMediaToSelf(
	v *events.Message,
	wac *whatsmeow.Client,
	selfJID types.JID,
	isManual bool,
) {

	msg := v.Message

	label := "🤖 Auto Saved"
	if isManual {
		label = "💾 Manually Saved"
	}

	// IMAGE
	if img := msg.GetImageMessage(); img != nil {
		if !isManual && !img.GetViewOnce() {
			return
		}
		data, err := wac.Download(context.Background(), img)
		if err != nil {
			log.Printf("[autosave] image download failed: %v", err)
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaImage)
		if err != nil {
			log.Printf("[autosave] image upload failed: %v", err)
			return
		}
		caption := fmt.Sprintf("%s\nFrom: *%s*", label, v.Info.PushName)
		ts := time.Now().Unix()
		_, err = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				Caption:           &caption,
				Mimetype:          img.Mimetype,
				URL:               &uploaded.URL,
				DirectPath:        &uploaded.DirectPath,
				MediaKey:          uploaded.MediaKey,
				FileEncSHA256:     uploaded.FileEncSHA256,
				FileSHA256:        uploaded.FileSHA256,
				FileLength:        &uploaded.FileLength,
				MediaKeyTimestamp: &ts,
				JPEGThumbnail:     img.JPEGThumbnail,
				Height:            img.Height,
				Width:             img.Width,
			},
		})
		if err != nil {
			log.Printf("[autosave] image send failed: %v", err)
		}
		return
	}

	// VIDEO
	if vid := msg.GetVideoMessage(); vid != nil {
		if !isManual && !vid.GetViewOnce() {
			return
		}
		data, err := wac.Download(context.Background(), vid)
		if err != nil {
			log.Printf("[autosave] video download failed: %v", err)
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaVideo)
		if err != nil {
			log.Printf("[autosave] video upload failed: %v", err)
			return
		}
		caption := fmt.Sprintf("%s\nFrom: *%s*", label, v.Info.PushName)
		ts := time.Now().Unix()
		_, err = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
			VideoMessage: &waProto.VideoMessage{
				Caption:           &caption,
				Mimetype:          vid.Mimetype,
				URL:               &uploaded.URL,
				DirectPath:        &uploaded.DirectPath,
				MediaKey:          uploaded.MediaKey,
				FileEncSHA256:     uploaded.FileEncSHA256,
				FileSHA256:        uploaded.FileSHA256,
				FileLength:        &uploaded.FileLength,
				Seconds:           vid.Seconds,
				MediaKeyTimestamp: &ts,
				JPEGThumbnail:     vid.JPEGThumbnail,
				Height:            vid.Height,
				Width:             vid.Width,
			},
		})
		if err != nil {
			log.Printf("[autosave] video send failed: %v", err)
		}
		return
	}

	// AUDIO
	if aud := msg.GetAudioMessage(); aud != nil {
		if !isManual && !aud.GetViewOnce() {
			return
		}
		data, err := wac.Download(context.Background(), aud)
		if err != nil {
			log.Printf("[autosave] audio download failed: %v", err)
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaAudio)
		if err != nil {
			log.Printf("[autosave] audio upload failed: %v", err)
			return
		}
		ts := time.Now().Unix()
		_, err = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
			AudioMessage: &waProto.AudioMessage{
				Mimetype:          aud.Mimetype,
				URL:               &uploaded.URL,
				DirectPath:        &uploaded.DirectPath,
				MediaKey:          uploaded.MediaKey,
				FileEncSHA256:     uploaded.FileEncSHA256,
				FileSHA256:        uploaded.FileSHA256,
				FileLength:        &uploaded.FileLength,
				Seconds:           aud.Seconds,
				MediaKeyTimestamp: &ts,
			},
		})
		if err != nil {
			log.Printf("[autosave] audio send failed: %v", err)
		}
		return
	}

	// DOCUMENT (no view-once concept for documents — manual .save only)
	if doc := msg.GetDocumentMessage(); doc != nil {
		if !isManual {
			return
		}
		data, err := wac.Download(context.Background(), doc)
		if err != nil {
			log.Printf("[autosave] document download failed: %v", err)
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaDocument)
		if err != nil {
			log.Printf("[autosave] document upload failed: %v", err)
			return
		}
		caption := fmt.Sprintf("%s\nFrom: *%s*", label, v.Info.PushName)
		ts := time.Now().Unix()
		_, err = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
			DocumentMessage: &waProto.DocumentMessage{
				Caption:           &caption,
				Mimetype:          doc.Mimetype,
				FileName:          doc.FileName,
				URL:               &uploaded.URL,
				DirectPath:        &uploaded.DirectPath,
				MediaKey:          uploaded.MediaKey,
				FileEncSHA256:     uploaded.FileEncSHA256,
				FileSHA256:        uploaded.FileSHA256,
				FileLength:        &uploaded.FileLength,
				MediaKeyTimestamp: &ts,
			},
		})
		if err != nil {
			log.Printf("[autosave] document send failed: %v", err)
		}
	}
}

// .autosave on/off
func autoSaveToggleHandler(
	v *events.Message,
	args []string,
) error {

	if len(args) == 0 {
		autoSaveMu.RLock()
		s := autoSaveEnabled
		autoSaveMu.RUnlock()
		state := "🟢 ON"
		if !s {
			state = "🔴 OFF"
		}
		return ReplyErr(v, fmt.Sprintf("💾 Auto Save: *%s*", state))
	}

	switch args[0] {
	case "on":
		autoSaveMu.Lock()
		autoSaveEnabled = true
		autoSaveMu.Unlock()
		return ReplyErr(v, "✅ Auto Save *ON*\nOnly view-once media will be auto-saved to Saved Messages (use *.save* on any replied message to save it manually).")
	case "off":
		autoSaveMu.Lock()
		autoSaveEnabled = false
		autoSaveMu.Unlock()
		return ReplyErr(v, "🔴 Auto Save *OFF*")
	}

	return ReplyErr(v, "❌ Use: .autosave on / off")
}

// .save — manually save replied media
func saveHandler(
	v *events.Message,
	args []string,
) error {

	quoted := GetQuotedMessage(v)
	if quoted == nil {
		return ReplyErr(v, "❌ Reply to a media message to save it.")
	}

	wac := client.Get()
	selfJID := getSelfJID()

	// Fake events.Message with quoted content
	fakeEvt := &events.Message{
		Info:    v.Info,
		Message: quoted,
	}

	saveMediaToSelf(fakeEvt, wac, selfJID, true)

	return ReplyErr(v, "✅ Media saved to *Saved Messages*!")
}

// REGISTER
func init() {

	Register(ModuleInfo{
		Name:        "AutoSave",
		Description: "Auto save media to self chat",

		Commands: []CommandInfo{

			{
				Pattern: "autosave",
				Handler: autoSaveToggleHandler,
				Sudo:    true,
			},

			{
				Pattern: "save",
				Handler: saveHandler,
				Sudo:    true,
			},
		},
	})
}
