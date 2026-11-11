package modules

// Newsletter (WhatsApp Channels) module — uses whatsmeow newsletter APIs.
//
// Commands:
//   .newsletter info <jid|invite>   — info about a channel
//   .newsletter follow <jid|invite> — follow/subscribe a channel
//   .newsletter unfollow <jid>      — unfollow a channel
//   .newsletter mute <jid>          — mute a channel
//   .newsletter unmute <jid>        — unmute a channel
//   .newsletter mylist              — list channels you follow
//   .newsletter create <name>       — create a new channel (requires business)
//   .newsletter send <jid> <text>   — send a text update to your channel

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"PbxGb/client"
)

// resolveNewsletterJID tries to parse a raw JID or fetch by invite link.
func resolveNewsletterJID(wac *whatsmeow.Client, arg string) (types.JID, error) {
	// newsletter JIDs end with @newsletter
	if strings.Contains(arg, "@newsletter") {
		return types.ParseJID(arg)
	}
	// treat as invite link / slug — fetch info first
	info, err := wac.GetNewsletterInfoWithInvite(context.Background(), arg)
	if err != nil {
		return types.EmptyJID, fmt.Errorf("can't resolve newsletter: %w", err)
	}
	return info.ID, nil
}

func newsletterHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage:\n"+
			"`.newsletter info <jid/invite>`\n"+
			"`.newsletter follow <jid/invite>`\n"+
			"`.newsletter unfollow <jid>`\n"+
			"`.newsletter mute <jid>`\n"+
			"`.newsletter unmute <jid>`\n"+
			"`.newsletter mylist`\n"+
			"`.newsletter create <name>`\n"+
			"`.newsletter send <jid> <text>`")
	}

	wac := client.Get()
	ctx := context.Background()
	sub := strings.ToLower(args[0])

	switch sub {
	case "info":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .newsletter info <jid/invite>")
		}
		arg := args[1]
		var info *types.NewsletterMetadata
		var err error
		if strings.Contains(arg, "@newsletter") {
			jid, jerr := types.ParseJID(arg)
			if jerr != nil {
				return ReplyErr(v, "❌ Invalid JID: "+arg)
			}
			info, err = wac.GetNewsletterInfo(ctx, jid)
		} else {
			info, err = wac.GetNewsletterInfoWithInvite(ctx, arg)
		}
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
		}
		state := "unknown"
		if info.ViewerMeta != nil {
			state = string(info.ViewerMeta.Role)
		}
		msg := fmt.Sprintf(
			"📢 *Newsletter Info*\n\n"+
				"📌 *Name:* %s\n"+
				"🆔 *JID:* `%s`\n"+
				"👥 *Subscribers:* %d\n"+
				"📝 *Desc:* %s\n"+
				"🔔 *Your role:* %s",
			info.ThreadMeta.Name.Text,
			info.ID.String(),
			info.ThreadMeta.SubscriberCount,
			info.ThreadMeta.Description.Text,
			state,
		)
		return ReplyErr(v, msg)

	case "follow", "subscribe":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .newsletter follow <jid/invite>")
		}
		jid, err := resolveNewsletterJID(wac, args[1])
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ %v", err))
		}
		if err := wac.FollowNewsletter(ctx, jid); err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Follow failed: %v", err))
		}
		return ReplyErr(v, fmt.Sprintf("✅ Followed channel: `%s`", jid.String()))

	case "unfollow", "unsubscribe":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .newsletter unfollow <jid>")
		}
		jid, err := types.ParseJID(args[1])
		if err != nil {
			return ReplyErr(v, "❌ Invalid JID: "+args[1])
		}
		if err := wac.UnfollowNewsletter(ctx, jid); err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Unfollow failed: %v", err))
		}
		return ReplyErr(v, fmt.Sprintf("✅ Unfollowed: `%s`", jid.String()))

	case "mute":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .newsletter mute <jid>")
		}
		jid, err := types.ParseJID(args[1])
		if err != nil {
			return ReplyErr(v, "❌ Invalid JID: "+args[1])
		}
		if err := wac.MuteNewsletter(ctx, jid, true); err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Mute failed: %v", err))
		}
		return ReplyErr(v, fmt.Sprintf("🔇 Muted: `%s`", jid.String()))

	case "unmute":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .newsletter unmute <jid>")
		}
		jid, err := types.ParseJID(args[1])
		if err != nil {
			return ReplyErr(v, "❌ Invalid JID: "+args[1])
		}
		if err := wac.MuteNewsletter(ctx, jid, false); err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Unmute failed: %v", err))
		}
		return ReplyErr(v, fmt.Sprintf("🔊 Unmuted: `%s`", jid.String()))

	case "mylist", "list", "following":
		newsletters, err := wac.GetSubscribedNewsletters(ctx)
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
		}
		if len(newsletters) == 0 {
			return ReplyErr(v, "ℹ️ You are not following any newsletters.")
		}
		lines := []string{fmt.Sprintf("📢 *Following %d newsletter(s):*\n", len(newsletters))}
		for i, n := range newsletters {
			muted := ""
			if n.ViewerMeta != nil && n.ViewerMeta.Mute == types.NewsletterMuteOn {
				muted = " 🔇"
			}
			lines = append(lines, fmt.Sprintf("%d. *%s*%s\n   `%s`", i+1, n.ThreadMeta.Name.Text, muted, n.ID.String()))
		}
		return ReplyErr(v, strings.Join(lines, "\n"))

	case "create":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .newsletter create <name>")
		}
		name := strings.Join(args[1:], " ")
		info, err := wac.CreateNewsletter(ctx, types.CreateNewsletterParams{
			Name: name,
		})
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Create failed: %v", err))
		}
		return ReplyErr(v, fmt.Sprintf(
			"✅ *Newsletter created!*\n📌 Name: %s\n🆔 JID: `%s`",
			info.ThreadMeta.Name.Text,
			info.ID.String(),
		))

	case "send":
		// .newsletter send <jid> <text...>
		if len(args) < 3 {
			return ReplyErr(v, "Usage: .newsletter send <jid> <text>")
		}
		jid, err := types.ParseJID(args[1])
		if err != nil {
			return ReplyErr(v, "❌ Invalid JID: "+args[1])
		}
		text := strings.Join(args[2:], " ")
		_, err = wac.SendMessage(ctx, jid, &waProto.Message{
			Conversation: proto.String(text),
		})
		if err != nil {
			return ReplyErr(v, fmt.Sprintf("❌ Send failed: %v", err))
		}
		return ReplyErr(v, "✅ Message sent to newsletter.")

	default:
		return ReplyErr(v, "❌ Unknown sub-command. Use: info / follow / unfollow / mute / unmute / mylist / create / send")
	}
}

func init() {
	Register(ModuleInfo{
		Name:        "Newsletter",
		Description: "WhatsApp Channels/Newsletters — follow, unfollow, mute, create, send",
		Commands: []CommandInfo{
			{Pattern: "newsletter", Handler: newsletterHandler, Sudo: true},
			{Pattern: "channel",    Handler: newsletterHandler, Sudo: true},
		},
	})
}
