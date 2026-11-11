package modules

// Privacy settings module — uses whatsmeow GetPrivacySettings / SetPrivacySettings
// and GetStatusPrivacy / SetStatusPrivacy.
//
// Commands:
//   .privacy              — show current privacy settings
//   .privacy lastseen contacts/everyone/nobody
//   .privacy pfp          contacts/everyone/nobody
//   .privacy status       contacts/everyone/nobody  (who can see status updates)
//   .privacy about        contacts/everyone/nobody
//   .privacy groups       contacts/everyone/admins
//   .privacy readreceipts on/off
//   .privacy calls        contacts/everyone/nobody
//   .privacy online       all/match_last_seen

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"PbxGb/client"
)

// privacySettingFromStr maps user-facing words to whatsmeow PrivacySetting constants.
func privacySettingFromStr(s string) (types.PrivacySetting, bool) {
	switch strings.ToLower(s) {
	case "all", "everyone":
		return types.PrivacySettingAll, true
	case "contacts":
		return types.PrivacySettingContacts, true
	case "contact_blacklist", "contacts_except":
		return types.PrivacySettingContactBlacklist, true
	case "nobody", "none":
		return types.PrivacySettingNone, true
	}
	return "", false
}

func privacySettingLabel(s types.PrivacySetting) string {
	switch s {
	case types.PrivacySettingAll:
		return "everyone"
	case types.PrivacySettingContacts:
		return "contacts"
	case types.PrivacySettingContactBlacklist:
		return "contacts_except"
	case types.PrivacySettingNone:
		return "nobody"
	}
	return string(s)
}

func privacyHandler(v *events.Message, args []string) error {
	wac := client.Get()
	ctx := context.Background()

	// No args → show current settings
	if len(args) == 0 {
		ps, err := wac.GetPrivacySettings(ctx)
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed to get privacy settings: %v", err))
		}
		msg := fmt.Sprintf(
			"╔══════════════════════╗\n"+
				"║   🔒 PRIVACY SETTINGS ║\n"+
				"╚══════════════════════╝\n\n"+
				"👁 *Last Seen:* %s\n"+
				"🖼 *Profile Photo:* %s\n"+
				"📝 *About:* %s\n"+
				"💬 *Status:* %s\n"+
				"👥 *Groups:* %s\n"+
				"📖 *Read Receipts:* %s\n"+
				"📞 *Calls:* %s\n"+
				"🟢 *Online:* %s",
			privacySettingLabel(ps.LastSeen),
			privacySettingLabel(ps.ProfilePhoto),
			privacySettingLabel(ps.About),
			privacySettingLabel(ps.Status),
			privacySettingLabel(ps.GroupAdd),
			func() string {
				if ps.ReadReceipts == types.PrivacySettingAll {
					return "on (everyone)"
				}
				return "off (nobody)"
			}(),
			privacySettingLabel(ps.CallAdd),
			string(ps.Online),
		)
		return ReplyErr(v, msg)
	}

	field := strings.ToLower(args[0])

	if len(args) < 2 {
		return ReplyErr(v, "❌ Usage: .privacy <field> <value>\nFields: lastseen, pfp, about, status, groups, readreceipts, calls, online")
	}
	val := strings.ToLower(args[1])

	switch field {
	case "lastseen", "last_seen":
		setting, ok := privacySettingFromStr(val)
		if !ok {
			return ReplyErr(v, "❌ Valid values: everyone / contacts / nobody")
		}
		if err := wac.SetPrivacySettings(ctx, types.PrivacyKeyLastSeen, setting); err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
		}
		return ReplyErr(v, fmt.Sprintf("✅ Last seen → *%s*", privacySettingLabel(setting)))

	case "pfp", "profilephoto", "pp":
		setting, ok := privacySettingFromStr(val)
		if !ok {
			return ReplyErr(v, "❌ Valid values: everyone / contacts / nobody")
		}
		if err := wac.SetPrivacySettings(ctx, types.PrivacyKeyProfilePhoto, setting); err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
		}
		return ReplyErr(v, fmt.Sprintf("✅ Profile photo → *%s*", privacySettingLabel(setting)))

	case "about", "bio":
		setting, ok := privacySettingFromStr(val)
		if !ok {
			return ReplyErr(v, "❌ Valid values: everyone / contacts / nobody")
		}
		if err := wac.SetPrivacySettings(ctx, types.PrivacyKeyAbout, setting); err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
		}
		return ReplyErr(v, fmt.Sprintf("✅ About → *%s*", privacySettingLabel(setting)))

	case "status":
		setting, ok := privacySettingFromStr(val)
		if !ok {
			return ReplyErr(v, "❌ Valid values: everyone / contacts / nobody")
		}
		if err := wac.SetPrivacySettings(ctx, types.PrivacyKeyStatus, setting); err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
		}
		return ReplyErr(v, fmt.Sprintf("✅ Status visibility → *%s*", privacySettingLabel(setting)))

	case "groups", "groupadd":
		setting, ok := privacySettingFromStr(val)
		if !ok {
			return ReplyErr(v, "❌ Valid values: everyone / contacts / nobody")
		}
		if err := wac.SetPrivacySettings(ctx, types.PrivacyKeyGroupAdd, setting); err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
		}
		return ReplyErr(v, fmt.Sprintf("✅ Group add → *%s*", privacySettingLabel(setting)))

	case "readreceipts", "readreceipt", "receipts":
		var setting types.PrivacySetting
		switch val {
		case "on", "enable":
			setting = types.PrivacySettingAll
		case "off", "disable":
			setting = types.PrivacySettingNone
		default:
			return ReplyErr(v, "❌ Valid values: on / off")
		}
		if err := wac.SetPrivacySettings(ctx, types.PrivacyKeyReadReceipts, setting); err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
		}
		label := "on"
		if setting == types.PrivacySettingNone {
			label = "off"
		}
		return ReplyErr(v, fmt.Sprintf("✅ Read receipts → *%s*", label))

	case "calls", "call":
		setting, ok := privacySettingFromStr(val)
		if !ok {
			return ReplyErr(v, "❌ Valid values: everyone / contacts / nobody")
		}
		if err := wac.SetPrivacySettings(ctx, types.PrivacyKeyCallAdd, setting); err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
		}
		return ReplyErr(v, fmt.Sprintf("✅ Calls → *%s*", privacySettingLabel(setting)))

	case "online":
		var setting types.PrivacySettingOnline
		switch val {
		case "all", "everyone":
			setting = types.PrivacySettingOnlineAll
		case "match", "match_last_seen", "matchlastseen":
			setting = types.PrivacySettingOnlineMatchLastSeen
		default:
			return ReplyErr(v, "❌ Valid values: all / match_last_seen")
		}
		if err := wac.SetPrivacySettings(ctx, types.PrivacyKeyOnline, types.PrivacySetting(setting)); err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
		}
		return ReplyErr(v, fmt.Sprintf("✅ Online visibility → *%s*", string(setting)))

	default:
		return ReplyErr(v, "❌ Unknown field. Use: lastseen / pfp / about / status / groups / readreceipts / calls / online")
	}
}

func init() {
	Register(ModuleInfo{
		Name:        "Privacy",
		Description: "Get/set WhatsApp privacy settings (last seen, pfp, about, groups, read receipts, calls, online)",
		Commands: []CommandInfo{
			{Pattern: "privacy", Handler: privacyHandler, Sudo: true},
		},
	})
}
