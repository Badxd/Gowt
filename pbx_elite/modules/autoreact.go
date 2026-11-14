package modules

// PBX Elite — Auto React Module
// ✅ Owner only / Sudo users
// ✅ Auto react to DM messages only
// ✅ Custom emoji or random from list
// ✅ Toggle on/off

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"PbxGb/client"
	"PbxGb/database"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	autoReactMu      sync.RWMutex
	autoReactEnabled = false
	autoReactEmojis  = []string{"❤️", "🔥", "😂", "👍", "🎉", "💯", "😍", "🙏"}
	autoReactCustom  = "" // if set, always use this emoji
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// HandleAutoReact — called from HandleMessage for every incoming DM
func HandleAutoReact(v *events.Message) {

	autoReactMu.RLock()
	enabled := autoReactEnabled
	autoReactMu.RUnlock()

	if !enabled {
		return
	}

	// DM only
	if v.Info.Chat.Server != types.DefaultUserServer {
		return
	}

	// Don't react to own messages
	wac := client.Get()
	if wac.Store.ID != nil &&
		v.Info.Sender.User == wac.Store.ID.User {
		return
	}

	// Owner & sudo bypass (react to everyone else's DMs)
	sender := v.Info.Sender.String()
	isPrivileged := isOwner(v) || database.IsSudo(sender)
	_ = isPrivileged // react to all DMs when enabled

	autoReactMu.RLock()
	custom := autoReactCustom
	emojis := autoReactEmojis
	autoReactMu.RUnlock()

	emoji := custom
	if emoji == "" {
		emoji = emojis[rng.Intn(len(emojis))]
	}

	// Send reaction
	_, _ = wac.SendMessage(
		context.Background(),
		v.Info.Chat,
		buildReaction(v.Info.ID, v.Info.Sender.String(), emoji),
	)
}

// .autoreact on/off
func autoReactToggleHandler(
	v *events.Message,
	args []string,
) error {

	if len(args) == 0 {
		autoReactMu.RLock()
		s := autoReactEnabled
		autoReactMu.RUnlock()

		state := "🟢 ON"
		if !s {
			state = "🔴 OFF"
		}
		return ReplyErr(v, fmt.Sprintf("⚡ Auto React DM: *%s*", state))
	}

	switch args[0] {
	case "on":
		autoReactMu.Lock()
		autoReactEnabled = true
		autoReactMu.Unlock()
		return ReplyErr(v, "✅ Auto React *ON*\nWill react to all DMs automatically.")

	case "off":
		autoReactMu.Lock()
		autoReactEnabled = false
		autoReactMu.Unlock()
		return ReplyErr(v, "🔴 Auto React *OFF*")
	}

	return ReplyErr(v, "❌ Use: .autoreact on / off")
}

// .reactemoji <emoji>  — set fixed emoji, or "random" to reset
func reactEmojiHandler(
	v *events.Message,
	args []string,
) error {

	if len(args) == 0 {
		autoReactMu.RLock()
		c := autoReactCustom
		autoReactMu.RUnlock()

		if c == "" {
			return ReplyErr(v, "🎲 Auto React: using *random* emoji from list")
		}
		return ReplyErr(v, fmt.Sprintf("😀 Auto React emoji: *%s*", c))
	}

	if args[0] == "random" {
		autoReactMu.Lock()
		autoReactCustom = ""
		autoReactMu.Unlock()
		return ReplyErr(v, "🎲 Auto React set to *random* emoji")
	}

	autoReactMu.Lock()
	autoReactCustom = args[0]
	autoReactMu.Unlock()

	return ReplyErr(v, fmt.Sprintf("✅ Auto React emoji set to: *%s*", args[0]))
}

// REGISTER
func init() {

	Register(ModuleInfo{
		Name:        "AutoReact",
		Description: "Auto react to DM messages",

		Commands: []CommandInfo{

			{
				Pattern: "autoreact",
				Handler: autoReactToggleHandler,
				Sudo:    true,
			},

			{
				Pattern: "reactemoji",
				Handler: reactEmojiHandler,
				Sudo:    true,
			},
		},
	})
}
