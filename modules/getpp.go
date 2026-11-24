package modules

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// .getpp [@user | number]  — get profile picture of a user
// .pfp  [@user | number]   — alias

func getppHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	var targetJID types.JID

	// 1. Mentioned user (@tag in message)
	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		if ci := ext.GetContextInfo(); ci != nil {
			for _, mention := range ci.GetMentionedJID() {
				j, err := types.ParseJID(mention)
				if err == nil {
					targetJID = j
					break
				}
			}
		}
	}

	// 2. Phone number in args
	if targetJID.IsEmpty() && len(args) > 0 {
		num := args[0]
		num = strings.TrimPrefix(num, "+")
		num = strings.TrimPrefix(num, "@")
		// strip non-digits
		clean := ""
		for _, c := range num {
			if c >= '0' && c <= '9' {
				clean += string(c)
			}
		}
		if clean != "" {
			j, err := types.ParseJID(clean + "@s.whatsapp.net")
			if err == nil {
				targetJID = j
			}
		}
	}

	// 3. Self
	if targetJID.IsEmpty() {
		if wac.Store.ID != nil {
			targetJID = types.NewJID(wac.Store.ID.User, types.DefaultUserServer)
		} else {
			return ReplyErr(v, "❌ Cannot determine your JID.")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Try full-size first, then preview fallback
	pic, err := wac.GetProfilePictureInfo(ctx, targetJID, &whatsmeow.GetProfilePictureParams{
		Preview: false,
	})
	if err != nil || pic == nil || pic.URL == "" {
		// Try preview (lower res, works even with some privacy settings)
		pic, err = wac.GetProfilePictureInfo(ctx, targetJID, &whatsmeow.GetProfilePictureParams{
			Preview: true,
		})
	}

	if err != nil {
		if strings.Contains(err.Error(), "not-authorized") || strings.Contains(err.Error(), "401") {
			return ReplyErr(v, "🔒 Profile picture is private (privacy settings restrict access).")
		}
		if strings.Contains(err.Error(), "item-not-found") || strings.Contains(err.Error(), "404") {
			return ReplyErr(v, "❌ No profile picture set.")
		}
		return ReplyErr(v, fmt.Sprintf("❌ Failed to get profile picture: %v", err))
	}

	if pic == nil || pic.URL == "" {
		return ReplyErr(v, "❌ No profile picture found (privacy may be enabled).")
	}

	// Download the image and send it
	imgCtx, imgCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer imgCancel()

	req, err := http.NewRequestWithContext(imgCtx, http.MethodGet, pic.URL, nil)
	if err == nil {
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			defer resp.Body.Close()
			imgData, err := io.ReadAll(resp.Body)
			if err == nil && len(imgData) > 0 {
				uploaded, err := wac.Upload(context.Background(), imgData, whatsmeow.MediaImage)
				if err == nil {
					mime := "image/jpeg"
					caption := fmt.Sprintf("🖼️ Profile picture of *%s*", targetJID.User)
					_, _ = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
						ImageMessage: &waProto.ImageMessage{
							Caption:       &caption,
							Mimetype:      &mime,
							URL:           &uploaded.URL,
							DirectPath:    &uploaded.DirectPath,
							MediaKey:      uploaded.MediaKey,
							FileEncSHA256: uploaded.FileEncSHA256,
							FileSHA256:    uploaded.FileSHA256,
							FileLength:    &uploaded.FileLength,
						},
					})
					return nil
				}
			}
		}
	}

	// Fallback: just send the URL
	return ReplyErr(v, fmt.Sprintf("🖼️ Profile picture:\n%s", pic.URL))
}

func init() {
	Register(ModuleInfo{
		Name:        "GetPP",
		Description: "Fetch and send profile picture of a user",
		Commands: []CommandInfo{
			{Pattern: "getpp", Handler: getppHandler, Sudo: true},
			{Pattern: "pfp", Handler: getppHandler, Sudo: true},
		},
	})
}
