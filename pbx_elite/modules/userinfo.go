package modules

import (
	"context"
	"fmt"
	"strings"
	"time"


	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// .userinfo @mention / number  — show detailed info about a WhatsApp user

func userInfoHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	// resolve target JID
	targetStr, err := extractTargetJID(v, args)
	if err != nil || targetStr == "" {
		// fall back to self
		if wac.Store.ID != nil {
			targetStr = wac.Store.ID.String()
		} else {
			return ReplyErr(v, "Usage: .userinfo @mention or .userinfo <number>")
		}
	}

	targetJID, err := types.ParseJID(targetStr)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Invalid JID: %v", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// fetch user info
	infos, err := wac.GetUserInfo(ctx, []string{targetJID.String()})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to get user info: %v", err))
	}

	phone := targetJID.User
	lines := []string{fmt.Sprintf("👤 *User Info: +%s*", phone), ""}

	for _, info := range infos {
		if info.VerifiedName != nil {
			lines = append(lines, fmt.Sprintf("✅ Verified: *%s*", info.VerifiedName.Name))
		}
		if info.PushName != "" {
			lines = append(lines, fmt.Sprintf("📛 Name: %s", info.PushName))
		}
		if info.Status != "" {
			lines = append(lines, fmt.Sprintf("📝 About: _%s_", info.Status))
		}
		if !info.LastSeen.IsZero() {
			lines = append(lines, fmt.Sprintf("🕒 Last seen: %s", info.LastSeen.Format("02 Jan 2006, 15:04")))
		}
		break
	}

	// profile picture
	pic, err := wac.GetProfilePictureInfo(ctx, targetJID, &types.GetProfilePictureParams{Preview: false})
	if err == nil && pic != nil {
		lines = append(lines, fmt.Sprintf("🖼️ [Profile Pic](%s)", pic.URL))
	} else {
		lines = append(lines, "🖼️ No profile picture")
	}

	lines = append(lines, "", fmt.Sprintf("📱 JID: `%s`", targetJID.String()))

	return ReplyErr(v, strings.Join(lines, "\n"))
}

func init() {
	Register(ModuleInfo{
		Name:        "User Info",
		Description: "Get detailed information about a WhatsApp user",
		Commands: []CommandInfo{
			{Pattern: "userinfo", Handler: userInfoHandler, Sudo: true},
			{Pattern: "uinfo", Handler: userInfoHandler, Sudo: true},
		},
	})
}
