package modules

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// Media filter: auto-delete messages of specific media types in a group
// Supported types: image, video, audio, document, sticker, link
//
// .mediafilter add <type>    — block a media type
// .mediafilter del <type>    — unblock a media type
// .mediafilter list          — show blocked types for this chat
// .mediafilter clear         — clear all blocks

var (
	mediaFilterMu   sync.RWMutex
	mediaFilterMap  = make(map[string]map[string]bool) // chatJID -> set of blocked types
)

// CheckMediaFilter is exported hook — auto-deletes blocked media types.
func CheckMediaFilter(v *events.Message) {
	chat := v.Info.Chat.String()
	if !strings.Contains(chat, "@g.us") {
		return
	}

	mediaFilterMu.RLock()
	blocked, ok := mediaFilterMap[chat]
	mediaFilterMu.RUnlock()

	if !ok || len(blocked) == 0 {
		return
	}

	wac := getClientSafe()
	if wac == nil {
		return
	}

	// don't filter own messages
	if wac.Store.ID != nil && strings.HasPrefix(v.Info.Sender.String(), wac.Store.ID.User) {
		return
	}

	msgType := detectMediaType(v)
	if msgType == "" {
		return
	}

	mediaFilterMu.RLock()
	shouldDelete := blocked[msgType] || blocked["all"]
	mediaFilterMu.RUnlock()

	if !shouldDelete {
		return
	}

	ctx, cancel := newContext(8)
	defer cancel()
	_ = wac.RevokeMessage(ctx, v.Info.Chat, false, v.Info.ID, "", "")
}

func detectMediaType(v *events.Message) string {
	msg := v.Message
	if msg.GetImageMessage() != nil {
		return "image"
	}
	if msg.GetVideoMessage() != nil {
		return "video"
	}
	if msg.GetAudioMessage() != nil {
		return "audio"
	}
	if msg.GetDocumentMessage() != nil {
		return "document"
	}
	if msg.GetStickerMessage() != nil {
		return "sticker"
	}
	// link detection from text
	body := ""
	if t := msg.GetConversation(); t != "" {
		body = t
	} else if ext := msg.GetExtendedTextMessage(); ext != nil {
		body = ext.GetText()
	}
	if body != "" {
		lower := strings.ToLower(body)
		if strings.Contains(lower, "http://") || strings.Contains(lower, "https://") ||
			strings.Contains(lower, "www.") {
			return "link"
		}
	}
	return ""
}

var validMediaTypes = map[string]bool{
	"image":    true,
	"video":    true,
	"audio":    true,
	"document": true,
	"sticker":  true,
	"link":     true,
	"all":      true,
}

func mediaFilterHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ This command can only be used in a group.")
	}

	chat := v.Info.Chat.String()

	if len(args) == 0 {
		return ReplyErr(v, "Usage:\n.mediafilter add <type>\n.mediafilter del <type>\n.mediafilter list\n.mediafilter clear\n\nTypes: image, video, audio, document, sticker, link, all")
	}

	sub := strings.ToLower(args[0])

	switch sub {
	case "add":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .mediafilter add <type>")
		}
		mtype := strings.ToLower(args[1])
		if !validMediaTypes[mtype] {
			return ReplyErr(v, fmt.Sprintf("❌ Invalid type: %s\nValid: image, video, audio, document, sticker, link, all", mtype))
		}
		mediaFilterMu.Lock()
		if mediaFilterMap[chat] == nil {
			mediaFilterMap[chat] = make(map[string]bool)
		}
		mediaFilterMap[chat][mtype] = true
		mediaFilterMu.Unlock()
		return ReplyErr(v, fmt.Sprintf("✅ *%s* messages will now be auto-deleted in this group.", mtype))

	case "del", "remove":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .mediafilter del <type>")
		}
		mtype := strings.ToLower(args[1])
		mediaFilterMu.Lock()
		if mediaFilterMap[chat] != nil {
			delete(mediaFilterMap[chat], mtype)
		}
		mediaFilterMu.Unlock()
		return ReplyErr(v, fmt.Sprintf("✅ Filter removed for *%s*.", mtype))

	case "list":
		mediaFilterMu.RLock()
		blocked := mediaFilterMap[chat]
		mediaFilterMu.RUnlock()

		if len(blocked) == 0 {
			return ReplyErr(v, "ℹ️ No media filters active in this group.")
		}

		var types []string
		for t := range blocked {
			types = append(types, t)
		}
		return ReplyErr(v, fmt.Sprintf("🚫 *Blocked media types:*\n%s", strings.Join(types, ", ")))

	case "clear":
		mediaFilterMu.Lock()
		delete(mediaFilterMap, chat)
		mediaFilterMu.Unlock()
		return ReplyErr(v, "✅ All media filters cleared.")

	default:
		return ReplyErr(v, "Usage: .mediafilter add/del/list/clear <type>")
	}

	_ = time.Now() // keep import
}

func init() {
	Register(ModuleInfo{
		Name:        "Media Filter",
		Description: "Auto-delete specific media types (image/video/audio/etc.) in groups",
		Commands: []CommandInfo{
			{Pattern: "mediafilter", Handler: mediaFilterHandler, Sudo: true},
			{Pattern: "mfilter", Handler: mediaFilterHandler, Sudo: true},
		},
	})
}
