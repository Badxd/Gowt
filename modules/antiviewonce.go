package modules

// PBX Elite — Anti View Once Module
//
// Automatically intercepts ANY incoming view-once image
// or video and silently saves it to owner's Saved Messages.
// The sender never knows it was saved.
//
// Commands:
//   .antiviewonce on   — enable
//   .antiviewonce off  — disable
//   .antiviewonce      — check status

import (
	"context"
	"fmt"
	"sync"

	"PbxGb/client"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	antiVOMu      sync.RWMutex
	antiVOEnabled = false
)

// HandleAntiViewOnce — hooked in main.go for every *events.Message
func HandleAntiViewOnce(v *events.Message) {

	antiVOMu.RLock()
	enabled := antiVOEnabled
	antiVOMu.RUnlock()

	if !enabled {
		return
	}

	if v.Message == nil {
		return
	}

	wac := client.Get()
	if wac.Store.ID == nil {
		return
	}

	// Skip own messages
	if v.Info.Sender.User == wac.Store.ID.User {
		return
	}

	// getSelfJID() is defined in autosave.go — shared helper
	selfJID := getSelfJID()

	// ── VIEW ONCE IMAGE ──────────────────────────────────────────────────
	img := v.Message.GetImageMessage()
	if img != nil && img.GetViewOnce() {

		data, err := wac.Download(context.Background(), img)
		if err != nil {
			return
		}

		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaImage)
		if err != nil {
			return
		}

		caption := fmt.Sprintf(
			"👁️ *Anti-ViewOnce: Image*\n"+
				"📩 From: *%s*\n"+
				"💬 Chat: %s",
			v.Info.PushName,
			v.Info.Chat.String(),
		)

		// Save to SELF CHAT ONLY — silent, no reply to sender
		_, _ = wac.SendMessage(
			context.Background(),
			selfJID,
			&waProto.Message{
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
			},
		)
		return
	}

	// ── VIEW ONCE VIDEO ──────────────────────────────────────────────────
	vid := v.Message.GetVideoMessage()
	if vid != nil && vid.GetViewOnce() {

		data, err := wac.Download(context.Background(), vid)
		if err != nil {
			return
		}

		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaVideo)
		if err != nil {
			return
		}

		caption := fmt.Sprintf(
			"👁️ *Anti-ViewOnce: Video*\n"+
				"📩 From: *%s*\n"+
				"💬 Chat: %s",
			v.Info.PushName,
			v.Info.Chat.String(),
		)

		_, _ = wac.SendMessage(
			context.Background(),
			selfJID,
			&waProto.Message{
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
			},
		)
	}
}

// .antiviewonce on/off/status
func antiVOToggleHandler(
	v *events.Message,
	args []string,
) error {

	if len(args) == 0 {
		antiVOMu.RLock()
		s := antiVOEnabled
		antiVOMu.RUnlock()

		state := "🟢 ON"
		if !s {
			state = "🔴 OFF"
		}
		return ReplyErr(v, fmt.Sprintf("👁️ Anti View Once: *%s*\n\nView-once media is silently saved to your Saved Messages.", state))
	}

	switch args[0] {
	case "on":
		antiVOMu.Lock()
		antiVOEnabled = true
		antiVOMu.Unlock()
		return ReplyErr(v,
			"✅ *Anti View Once ON*\n\n"+
				"All incoming view-once images/videos will be\n"+
				"automatically saved to your *Saved Messages*.\n"+
				"Sender will NOT know.")

	case "off":
		antiVOMu.Lock()
		antiVOEnabled = false
		antiVOMu.Unlock()
		return ReplyErr(v, "🔴 *Anti View Once OFF*")
	}

	return ReplyErr(v, "❌ Use: .antiviewonce on / off")
}

// REGISTER
func init() {

	Register(ModuleInfo{
		Name:        "AntiViewOnce",
		Description: "Auto-save view-once media to Saved Messages",

		Commands: []CommandInfo{
			{
				Pattern: "antiviewonce",
				Handler: antiVOToggleHandler,
				Sudo:    true,
			},
		},
	})
}
