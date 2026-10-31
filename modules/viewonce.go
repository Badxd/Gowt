package modules

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"sync"
	"time"

	"PbxGb/client"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

// reencodeImage decodes+re-compresses image bytes so the re-uploaded copy
// has a different file hash than the original. WhatsApp appears to hold/
// flag re-uploads of view-once media when the bytes are byte-for-byte
// identical to the once-viewed original, showing "Waiting for this
// message" indefinitely on the receiving side. Re-encoding (same visual
// content, different compressed bytes) avoids that exact-match fingerprint.
// If decoding fails (unsupported format) it returns the original bytes
// unchanged as a best-effort fallback.
func reencodeImage(data []byte, mimetype string) ([]byte, string) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, mimetype
	}

	var buf bytes.Buffer
	switch format {
	case "png":
		if err := png.Encode(&buf, img); err != nil {
			return data, mimetype
		}
		return buf.Bytes(), "image/png"
	default: // jpeg and anything else we can decode
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 92}); err != nil {
			return data, mimetype
		}
		return buf.Bytes(), "image/jpeg"
	}
}

var (
	autoVVMu      sync.RWMutex
	autoVVEnabled = false // automatic view-once save
)

// HandleAutoVV — har incoming message check karo view-once ke liye
func HandleAutoVV(v *events.Message) {
	autoVVMu.RLock()
	enabled := autoVVEnabled
	autoVVMu.RUnlock()
	if !enabled {
		return
	}

	if v.Info.IsFromMe {
		return
	}

	wac := client.Get()
	if wac.Store.ID != nil && v.Info.Sender.User == wac.Store.ID.User {
		return
	}

	msg := v.Message
	if msg == nil {
		return
	}

	selfJID := getSelfJID()

	// View-once image
	if img := msg.GetImageMessage(); img != nil && img.GetViewOnce() {
		data, err := wac.Download(context.Background(), img)
		if err != nil {
			log.Printf("[autovv] image download failed: %v", err)
			return
		}
		mimetype := img.GetMimetype()
		data, reMime := reencodeImage(data, mimetype)
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaImage)
		if err != nil {
			log.Printf("[autovv] image upload failed: %v", err)
			return
		}
		caption := fmt.Sprintf("👁️ *Auto Saved View Once*\nFrom: *%s*", v.Info.PushName)
		ts := time.Now().Unix()
		_, err = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				Caption:           &caption,
				Mimetype:          &reMime,
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
			log.Printf("[autovv] image send failed: %v", err)
		}
		return
	}

	// View-once video
	if vid := msg.GetVideoMessage(); vid != nil && vid.GetViewOnce() {
		data, err := wac.Download(context.Background(), vid)
		if err != nil {
			log.Printf("[autovv] video download failed: %v", err)
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaVideo)
		if err != nil {
			log.Printf("[autovv] video upload failed: %v", err)
			return
		}
		caption := fmt.Sprintf("👁️ *Auto Saved View Once Video*\nFrom: *%s*", v.Info.PushName)
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
			log.Printf("[autovv] video send failed: %v", err)
		}
	}
}

// HandleEmojiVV — 🫣 (and a few other) emoji reply pe view-once/media save karo
// (no dot prefix needed). Owner/sudo only.
func HandleEmojiVV(v *events.Message) {
	// NOTE: do NOT gate on v.Info.IsFromMe here. The owner triggers this
	// by sending the emoji from their OWN phone/linked device, which is
	// exactly what IsFromMe=true means for a self-bot. isOwner() below
	// already treats IsFromMe as owner and authorizes it.
	if !isOwner(v) {
		return
	}

	text := extractText(v)

	allowedEmojis := map[string]bool{
		"🫣": true, "😂": true, "😭": true, "💞": true, "❤️": true,
		"🥹": true, "😍": true, "🥳": true, "😙": true, "🤥": true,
		"🌝": true, "🌚": true, "☠️": true, "😈": true, "👿": true,
		"🙃": true, "🙂": true,
	}

	if !allowedEmojis[text] {
		return
	}

	quoted := GetQuotedMessage(v)
	if quoted == nil {
		return
	}

	// Image save
	if img := quoted.GetImageMessage(); img != nil {
		wac := client.Get()
		data, err := wac.Download(context.Background(), img)
		if err != nil {
			log.Printf("[emojivv] image download failed: %v", err)
			return
		}
		mimetype := img.GetMimetype()
		data, reMime := reencodeImage(data, mimetype)
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaImage)
		if err != nil {
			log.Printf("[emojivv] image upload failed: %v", err)
			return
		}
		selfJID := getSelfJID()
		caption := fmt.Sprintf("%s *Saved via Emoji*\nFrom: *%s*\nChat: %s",
			text, v.Info.PushName, v.Info.Chat.String())
		ts := time.Now().Unix()
		_, err = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				Caption:           &caption,
				Mimetype:          &reMime,
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
			log.Printf("[emojivv] image send failed: %v", err)
		}
		return
	}

	// Video save
	if vid := quoted.GetVideoMessage(); vid != nil {
		wac := client.Get()
		data, err := wac.Download(context.Background(), vid)
		if err != nil {
			log.Printf("[emojivv] video download failed: %v", err)
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaVideo)
		if err != nil {
			log.Printf("[emojivv] video upload failed: %v", err)
			return
		}
		selfJID := getSelfJID()
		caption := fmt.Sprintf("%s *Saved via Emoji*\nFrom: *%s*\nChat: %s",
			text, v.Info.PushName, v.Info.Chat.String())
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
			log.Printf("[emojivv] video send failed: %v", err)
		}
	}
}

// .vv — command se save (reply karo view-once pe, koi confirm msg nahi aye ga)
func vvHandler(v *events.Message, args []string) error {
	quoted := GetQuotedMessage(v)
	if quoted == nil {
		return ReplyErr(v, "❌ Pehle kisi view-once image/video nu reply karo phir .vv likho")
	}

	if quoted.ImageMessage != nil {
		return saveVOImage(v, quoted.ImageMessage)
	}
	if quoted.VideoMessage != nil {
		return saveVOVideo(v, quoted.VideoMessage)
	}
	return ReplyErr(v, "❌ Replied message view-once nahi hai.")
}

func saveVOImage(v *events.Message, img *waProto.ImageMessage) error {
	wac := client.Get()
	data, err := wac.Download(context.Background(), img)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
	}
	mimetype := img.GetMimetype()
	data, reMime := reencodeImage(data, mimetype)
	uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaImage)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Upload failed: %v", err))
	}
	caption := fmt.Sprintf("👁️ *View Once Image*\nFrom: *%s*\nChat: %s", v.Info.PushName, v.Info.Chat.String())
	selfJID := getSelfJID()
	ts := time.Now().Unix()
	_, err = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			Caption:           &caption,
			Mimetype:          &reMime,
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
		log.Printf("[.vv] image send failed: %v", err)
		return ReplyErr(v, fmt.Sprintf("❌ Save failed: %v", err))
	}
	// Silent — koi confirm message nahi bheja original chat mein
	return nil
}

func saveVOVideo(v *events.Message, vid *waProto.VideoMessage) error {
	wac := client.Get()
	data, err := wac.Download(context.Background(), vid)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Download failed: %v", err))
	}
	uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaVideo)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Upload failed: %v", err))
	}
	caption := fmt.Sprintf("👁️ *View Once Video*\nFrom: *%s*\nChat: %s", v.Info.PushName, v.Info.Chat.String())
	selfJID := getSelfJID()
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
		log.Printf("[.vv] video send failed: %v", err)
		return ReplyErr(v, fmt.Sprintf("❌ Save failed: %v", err))
	}
	return nil
}

// .autovv on/off
func autoVVToggleHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		autoVVMu.RLock()
		s := autoVVEnabled
		autoVVMu.RUnlock()
		state := "🟢 ON"
		if !s {
			state = "🔴 OFF"
		}
		return ReplyErr(v, fmt.Sprintf("👁️ Auto View-Once Save: *%s*", state))
	}
	switch args[0] {
	case "on":
		autoVVMu.Lock()
		autoVVEnabled = true
		autoVVMu.Unlock()
		return ReplyErr(v, "✅ Auto View-Once Save *ON*\nSaari view-once media automatically saved messages vich save hogi.")
	case "off":
		autoVVMu.Lock()
		autoVVEnabled = false
		autoVVMu.Unlock()
		return ReplyErr(v, "🔴 Auto View-Once Save *OFF*")
	}
	return ReplyErr(v, "❌ Use: .autovv on / off")
}

func init() {
	Register(ModuleInfo{
		Name:        "ViewOnce",
		Description: "Save view-once media silently + auto save + 🫣 emoji save",
		Commands: []CommandInfo{
			{Pattern: "vv", Handler: vvHandler, Sudo: true},
			{Pattern: "autovv", Handler: autoVVToggleHandler, Sudo: true},
		},
	})
}
