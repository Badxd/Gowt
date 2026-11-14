package modules

import (
	"context"
	"fmt"
	"strings"
	"time"


	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// .online       — broadcast online presence
// .offline      — broadcast offline/unavailable presence
// .typing <n>   — simulate typing in current chat for n seconds (default 3)
// .recording    — simulate recording audio in current chat

func onlineHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := wac.SendPresence(ctx, types.PresenceAvailable)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, "🟢 Presence set to *online*.")
}

func offlineHandler(v *events.Message, args []string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := wac.SendPresence(ctx, types.PresenceUnavailable)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, "⚫ Presence set to *offline*.")
}

func typingHandler(v *events.Message, args []string) error {
	duration := 3 * time.Second
	if len(args) > 0 {
		var secs int
		if _, err := fmt.Sscan(args[0], &secs); err == nil && secs > 0 && secs <= 60 {
			duration = time.Duration(secs) * time.Second
		}
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Send composing state
	err := wac.SendChatPresence(ctx, v.Info.Chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	// schedule paused state after duration
	go func() {
		time.Sleep(duration)
		ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel2()
		_ = wac.SendChatPresence(ctx2, v.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaText)
	}()

	return ReplyErr(v, fmt.Sprintf("⌨️ Simulating typing for *%s*…", duration.Round(time.Second)))
}

func recordingHandler(v *events.Message, args []string) error {
	duration := 5 * time.Second
	if len(args) > 0 {
		var secs int
		if _, err := fmt.Sscan(args[0], &secs); err == nil && secs > 0 && secs <= 60 {
			duration = time.Duration(secs) * time.Second
		}
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := wac.SendChatPresence(ctx, v.Info.Chat, types.ChatPresenceComposing, types.ChatPresenceMediaAudio)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	go func() {
		time.Sleep(duration)
		ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel2()
		_ = wac.SendChatPresence(ctx2, v.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaAudio)
	}()

	return ReplyErr(v, fmt.Sprintf("🎙️ Simulating recording for *%s*…", duration.Round(time.Second)))
}

// fakeLastSeen — set last seen privacy
func fakeLastSeenHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .lastseen everyone/contacts/nobody")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var ps types.PrivacySetting
	switch strings.ToLower(args[0]) {
	case "everyone", "all":
		ps = types.PrivacySettingAll
	case "contacts":
		ps = types.PrivacySettingContacts
	case "nobody", "none":
		ps = types.PrivacySettingNone
	default:
		return ReplyErr(v, "Usage: .lastseen everyone/contacts/nobody")
	}

	_, err := wac.SetPrivacySetting(ctx, types.PrivacySettingTypeLastSeen, ps)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	return ReplyErr(v, fmt.Sprintf("✅ Last seen visibility set to *%s*.", args[0]))
}

func init() {
	Register(ModuleInfo{
		Name:        "Presence",
		Description: "Fake online/offline/typing/recording presence",
		Commands: []CommandInfo{
			{Pattern: "online", Handler: onlineHandler, Sudo: true},
			{Pattern: "offline", Handler: offlineHandler, Sudo: true},
			{Pattern: "typing", Handler: typingHandler, Sudo: true},
			{Pattern: "recording", Handler: recordingHandler, Sudo: true},
			{Pattern: "lastseen", Handler: fakeLastSeenHandler, Sudo: true},
		},
	})
}
