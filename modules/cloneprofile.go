package modules

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"PbxGb/client"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// getMentionedJIDs extracts @-mentioned JIDs from a message
func getMentionedJIDs(v *events.Message) []types.JID {
	var result []types.JID
	var contextInfo interface{ GetMentionedJid() []string }

	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		contextInfo = ext.GetContextInfo()
	}
	if contextInfo == nil {
		return result
	}
	for _, raw := range contextInfo.GetMentionedJid() {
		jid, err := types.ParseJID(raw)
		if err == nil {
			result = append(result, jid)
		}
	}
	return result
}

// downloadBytes fetches bytes from a URL with a timeout
func downloadBytes(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// .setpfp — set own profile picture from attached/replied image or URL
func setpfpHandler(v *events.Message, args []string) error {
	wac := client.Get()

	var imgBytes []byte
	var err error

	// Attached image
	if img := v.Message.GetImageMessage(); img != nil {
		imgBytes, err = wac.Download(img)
		if err != nil {
			return ReplyErr(v, "❌ Failed to download image: "+err.Error())
		}
	}

	// Quoted image
	if imgBytes == nil {
		if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			if ctx := ext.GetContextInfo(); ctx != nil {
				if qm := ctx.GetQuotedMessage(); qm != nil {
					if qimg := qm.GetImageMessage(); qimg != nil {
						imgBytes, err = wac.Download(qimg)
						if err != nil {
							return ReplyErr(v, "❌ Failed to download quoted image: "+err.Error())
						}
					}
				}
			}
		}
	}

	// URL argument
	if imgBytes == nil && len(args) > 0 {
		imgBytes, err = downloadBytes(args[0])
		if err != nil {
			return ReplyErr(v, "❌ Failed to download from URL: "+err.Error())
		}
	}

	if len(imgBytes) == 0 {
		return ReplyErr(v, "Usage: .setpfp — attach/reply to an image, or provide a URL\nExample: .setpfp https://example.com/photo.jpg")
	}

	selfJID := wac.Store.ID.ToNonAD()
	_, err = wac.SetGroupPhoto(context.Background(), selfJID, imgBytes)
	if err != nil {
		return ReplyErr(v, "❌ Failed to set profile picture: "+err.Error())
	}
	return ReplyErr(v, "✅ Profile picture updated!")
}

// .clone @user — clone profile picture from another user
func cloneFullHandler(v *events.Message, args []string) error {
	wac := client.Get()

	var targetJID types.JID
	var err error

	// Check @mention
	mentioned := getMentionedJIDs(v)
	if len(mentioned) > 0 {
		targetJID = mentioned[0]
	} else if len(args) > 0 {
		raw := strings.TrimPrefix(args[0], "@")
		raw = strings.ReplaceAll(raw, "-", "")
		if !strings.Contains(raw, "@") {
			raw += "@s.whatsapp.net"
		}
		targetJID, err = types.ParseJID(raw)
		if err != nil {
			return ReplyErr(v, "❌ Invalid user: "+args[0])
		}
	} else {
		return ReplyErr(v, "Usage: .clone @user\nClones profile picture from that user.")
	}

	picInfo, err := wac.GetProfilePictureInfo(context.Background(), targetJID, &types.GetProfilePictureParams{})
	if err != nil || picInfo == nil {
		return ReplyErr(v, "❌ Could not fetch profile picture (private or not found).")
	}

	imgBytes, err := downloadBytes(picInfo.URL)
	if err != nil {
		return ReplyErr(v, "❌ Failed to download profile picture: "+err.Error())
	}

	selfJID := wac.Store.ID.ToNonAD()
	_, err = wac.SetGroupPhoto(context.Background(), selfJID, imgBytes)
	if err != nil {
		return ReplyErr(v, "❌ Failed to set profile picture: "+err.Error())
	}

	return ReplyErr(v, "✅ Profile picture cloned from @"+targetJID.User+"!")
}

func init() {
	Register(ModuleInfo{
		Name:        "CloneProfile",
		Description: "Clone profile picture from another user, or set your own PFP",
		Commands: []CommandInfo{
			{Pattern: "setpfp", Handler: setpfpHandler, Sudo: true},
			{Pattern: "clone", Handler: cloneFullHandler, Sudo: true},
		},
	})
}
