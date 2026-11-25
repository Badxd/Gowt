package modules

// Media Filter — per-chat keyword→media storage.
//
// Usage:
//   .filter add <keyword>   — reply to a media message to store it under that keyword
//   .filter del <keyword>   — remove a stored filter
//   .filter list            — show all stored filters for this chat
//   .filter clear           — clear all filters in this chat
//
// When someone types .<keyword> alone in the chat (e.g. ".sukh"), the bot
// resends the stored media for that keyword.
//
// ALSO keeps the original word-ban behaviour as ".filterban" to avoid breaking things.
//   .filterban add <word>   — ban word (auto-delete messages containing it)
//   .filterban del <word>
//   .filterban list
//   .filterban on/off

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// ─── MEDIA FILTER STORE ──────────────────────────────────────────────────────

type mediaFilterEntry struct {
	Kind      string // "image" | "video" | "audio" | "document"
	MimeType  string
	FileData  []byte // downloaded raw bytes
	Caption   string
}

var (
	kwFilterMu    sync.RWMutex
	kwFilterStore = make(map[string]map[string]*mediaFilterEntry) // chatJID → keyword → entry
)

// CheckKeywordFilter is the exported hook called from HandleMessage.
// When someone sends ".<keyword>" and that keyword has a stored media,
// the bot resends that media.
func CheckKeywordFilter(v *events.Message) {
	chat := v.Info.Chat.String()

	kwFilterMu.RLock()
	chatMap := kwFilterStore[chat]
	kwFilterMu.RUnlock()

	if len(chatMap) == 0 {
		return
	}

	// get text body
	body := strings.TrimSpace(extractBodyText(v))
	if body == "" || !strings.HasPrefix(body, ".") {
		return
	}
	keyword := strings.ToLower(strings.TrimPrefix(body, "."))
	if keyword == "" {
		return
	}

	kwFilterMu.RLock()
	entry, ok := kwFilterStore[chat][keyword]
	kwFilterMu.RUnlock()

	if !ok || entry == nil {
		return
	}

	wac := getClientSafe()
	if wac == nil {
		return
	}

	go func() {
		_ = sendMediaFilterEntry(wac, v, entry)
	}()
}

func sendMediaFilterEntry(wac *whatsmeow.Client, v *events.Message, entry *mediaFilterEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch entry.Kind {
	case "image":
		uploaded, err := wac.Upload(ctx, entry.FileData, whatsmeow.MediaImage)
		if err != nil {
			return err
		}
		mime := entry.MimeType
		if mime == "" {
			mime = "image/jpeg"
		}
		cap := entry.Caption
		fl := uploaded.FileLength
		_, err = wac.SendMessage(ctx, v.Info.Chat, &waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &fl,
				Mimetype:      &mime,
				Caption:       &cap,
			},
		})
		return err

	case "video":
		uploaded, err := wac.Upload(ctx, entry.FileData, whatsmeow.MediaVideo)
		if err != nil {
			return err
		}
		mime := entry.MimeType
		if mime == "" {
			mime = "video/mp4"
		}
		cap := entry.Caption
		fl := uploaded.FileLength
		_, err = wac.SendMessage(ctx, v.Info.Chat, &waE2E.Message{
			VideoMessage: &waE2E.VideoMessage{
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &fl,
				Mimetype:      &mime,
				Caption:       &cap,
			},
		})
		return err

	case "audio":
		uploaded, err := wac.Upload(ctx, entry.FileData, whatsmeow.MediaAudio)
		if err != nil {
			return err
		}
		mime := entry.MimeType
		if mime == "" {
			mime = "audio/mp4"
		}
		fl := uploaded.FileLength
		ptt := false
		_, err = wac.SendMessage(ctx, v.Info.Chat, &waE2E.Message{
			AudioMessage: &waE2E.AudioMessage{
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &fl,
				Mimetype:      &mime,
				PTT:           &ptt,
			},
		})
		return err

	case "document":
		uploaded, err := wac.Upload(ctx, entry.FileData, whatsmeow.MediaDocument)
		if err != nil {
			return err
		}
		mime := entry.MimeType
		if mime == "" {
			mime = "application/octet-stream"
		}
		fl := uploaded.FileLength
		cap := entry.Caption
		_, err = wac.SendMessage(ctx, v.Info.Chat, &waE2E.Message{
			DocumentMessage: &waE2E.DocumentMessage{
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &fl,
				Mimetype:      &mime,
				Caption:       &cap,
			},
		})
		return err
	}
	return nil
}

// filterHandler — .filter add/del/list/clear
func filterHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()

	if len(args) == 0 {
		return ReplyErr(v, "📂 *Media Filter Commands:*\n"+
			"`.filter add <keyword>` — reply to media to store it\n"+
			"`.filter del <keyword>` — delete stored media\n"+
			"`.filter list` — show all stored keywords\n"+
			"`.filter clear` — clear all filters\n\n"+
			"To trigger: type `.<keyword>` in chat")
	}

	sub := strings.ToLower(args[0])

	switch sub {
	case "add":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .filter add <keyword>\n(reply to a media message)")
		}
		keyword := strings.ToLower(strings.Join(args[1:], " "))
		if strings.ContainsAny(keyword, " \t\n") {
			return ReplyErr(v, "❌ Keyword should not contain spaces. Use underscore: e.g. `my_pic`")
		}

		// get quoted/replied media
		wac := getClientSafe()
		if wac == nil {
			return ReplyErr(v, "❌ Client not available.")
		}

		entry, err := downloadMediaFromReply(v, wac)
		if err != nil || entry == nil {
			return ReplyErr(v, "❌ Reply to an image/video/audio/document to store it as a filter.\n"+
				"Example: reply to an image + `.filter add sukh`")
		}

		kwFilterMu.Lock()
		if kwFilterStore[chat] == nil {
			kwFilterStore[chat] = make(map[string]*mediaFilterEntry)
		}
		kwFilterStore[chat][keyword] = entry
		kwFilterMu.Unlock()

		return ReplyErr(v, fmt.Sprintf("✅ Filter *%s* saved.\nType `.%s` anytime to resend this media.", keyword, keyword))

	case "del", "remove", "delete":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .filter del <keyword>")
		}
		keyword := strings.ToLower(strings.Join(args[1:], " "))
		kwFilterMu.Lock()
		if kwFilterStore[chat] != nil {
			delete(kwFilterStore[chat], keyword)
		}
		kwFilterMu.Unlock()
		return ReplyErr(v, fmt.Sprintf("✅ Filter *%s* removed.", keyword))

	case "list":
		kwFilterMu.RLock()
		chatMap := kwFilterStore[chat]
		kwFilterMu.RUnlock()

		if len(chatMap) == 0 {
			return ReplyErr(v, "ℹ️ No media filters saved in this chat.")
		}
		var lines []string
		lines = append(lines, fmt.Sprintf("📂 *Media Filters (%d):*", len(chatMap)))
		for kw, entry := range chatMap {
			lines = append(lines, fmt.Sprintf("• `.%s` → %s", kw, entry.Kind))
		}
		return ReplyErr(v, strings.Join(lines, "\n"))

	case "clear":
		kwFilterMu.Lock()
		delete(kwFilterStore, chat)
		kwFilterMu.Unlock()
		return ReplyErr(v, "✅ All media filters cleared for this chat.")

	default:
		return ReplyErr(v, "Usage: .filter add/del/list/clear")
	}
}

// downloadMediaFromReply downloads the media from a quoted/replied message.
func downloadMediaFromReply(v *events.Message, wac *whatsmeow.Client) (*mediaFilterEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	qm := GetQuotedMessage(v)
	if qm == nil {
		// also check direct attachment
		qm = v.Message
	}
	if qm == nil {
		return nil, fmt.Errorf("no message")
	}

	if img := qm.GetImageMessage(); img != nil {
		data, err := wac.Download(ctx, img)
		if err != nil {
			return nil, err
		}
		mime := img.GetMimetype()
		if mime == "" {
			mime = "image/jpeg"
		}
		return &mediaFilterEntry{
			Kind:     "image",
			MimeType: mime,
			FileData: data,
			Caption:  img.GetCaption(),
		}, nil
	}

	if vid := qm.GetVideoMessage(); vid != nil {
		data, err := wac.Download(ctx, vid)
		if err != nil {
			return nil, err
		}
		return &mediaFilterEntry{
			Kind:     "video",
			MimeType: vid.GetMimetype(),
			FileData: data,
			Caption:  vid.GetCaption(),
		}, nil
	}

	if aud := qm.GetAudioMessage(); aud != nil {
		data, err := wac.Download(ctx, aud)
		if err != nil {
			return nil, err
		}
		return &mediaFilterEntry{
			Kind:     "audio",
			MimeType: aud.GetMimetype(),
			FileData: data,
		}, nil
	}

	if doc := qm.GetDocumentMessage(); doc != nil {
		data, err := wac.Download(ctx, doc)
		if err != nil {
			return nil, err
		}
		return &mediaFilterEntry{
			Kind:     "document",
			MimeType: doc.GetMimetype(),
			FileData: data,
			Caption:  doc.GetCaption(),
		}, nil
	}

	return nil, fmt.Errorf("no supported media in replied message")
}

// ─── WORD BAN FILTER (original behaviour, now under .filterban) ──────────────

var (
	filterBanMu      sync.RWMutex
	filterBanWords   = make(map[string][]string) // chatJID -> []banned words
	filterBanEnabled = make(map[string]bool)
)

// CheckFilters is called from HandleMessage — auto-deletes messages with banned words.
func CheckFilters(v *events.Message) {
	chat := v.Info.Chat.String()

	filterBanMu.RLock()
	enabled := filterBanEnabled[chat]
	words := filterBanWords[chat]
	filterBanMu.RUnlock()

	if !enabled || len(words) == 0 {
		return
	}

	wac := getClientSafe()
	if wac == nil {
		return
	}
	if wac.Store.ID != nil && strings.HasPrefix(v.Info.Sender.String(), wac.Store.ID.User) {
		return
	}

	body := strings.ToLower(extractBodyText(v))
	if body == "" {
		return
	}

	matched := ""
	for _, w := range words {
		if strings.Contains(body, strings.ToLower(w)) {
			matched = w
			break
		}
	}
	if matched == "" {
		return
	}

	ctx, cancel := newContext(8)
	defer cancel()
	_, _ = wac.RevokeMessage(ctx, v.Info.Chat, v.Info.ID)

	mentions := []string{v.Info.Sender.String()}
	warn := fmt.Sprintf("⚠️ @%s, your message was removed (banned word).", v.Info.Sender.User)
	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(warn),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: mentions,
			},
		},
	}
	ctx2, cancel2 := newContext(8)
	defer cancel2()
	_, _ = wac.SendMessage(ctx2, v.Info.Chat, msg)
}

func filterBanHandler(v *events.Message, args []string) error {
	if !strings.Contains(v.Info.Chat.String(), "@g.us") {
		return ReplyErr(v, "❌ Groups only.")
	}
	chat := v.Info.Chat.String()

	if len(args) == 0 {
		return ReplyErr(v, "Usage:\n.filterban add <word>\n.filterban del <word>\n.filterban list\n.filterban on/off")
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "add":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .filterban add <word>")
		}
		word := strings.ToLower(strings.Join(args[1:], " "))
		filterBanMu.Lock()
		for _, w := range filterBanWords[chat] {
			if w == word {
				filterBanMu.Unlock()
				return ReplyErr(v, fmt.Sprintf("ℹ️ *%s* is already banned.", word))
			}
		}
		filterBanWords[chat] = append(filterBanWords[chat], word)
		filterBanEnabled[chat] = true
		filterBanMu.Unlock()
		return ReplyErr(v, fmt.Sprintf("✅ *%s* added to ban list.", word))

	case "del", "remove":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .filterban del <word>")
		}
		word := strings.ToLower(strings.Join(args[1:], " "))
		filterBanMu.Lock()
		list := filterBanWords[chat]
		var newList []string
		found := false
		for _, w := range list {
			if w == word {
				found = true
				continue
			}
			newList = append(newList, w)
		}
		filterBanWords[chat] = newList
		filterBanMu.Unlock()
		if !found {
			return ReplyErr(v, fmt.Sprintf("❌ *%s* was not in the ban list.", word))
		}
		return ReplyErr(v, fmt.Sprintf("✅ *%s* removed from ban list.", word))

	case "list":
		filterBanMu.RLock()
		list := filterBanWords[chat]
		enabled := filterBanEnabled[chat]
		filterBanMu.RUnlock()
		if len(list) == 0 {
			return ReplyErr(v, "ℹ️ No banned words.")
		}
		status := "🔴 off"
		if enabled {
			status = "🟢 on"
		}
		return ReplyErr(v, fmt.Sprintf("🚫 *Banned words (%s):*\n%s", status, strings.Join(list, ", ")))

	case "on", "enable":
		filterBanMu.Lock()
		filterBanEnabled[chat] = true
		filterBanMu.Unlock()
		return ReplyErr(v, "✅ Word ban filter *enabled*.")

	case "off", "disable":
		filterBanMu.Lock()
		filterBanEnabled[chat] = false
		filterBanMu.Unlock()
		return ReplyErr(v, "✅ Word ban filter *disabled*.")

	default:
		return ReplyErr(v, "Usage: .filterban add/del/list/on/off")
	}
}

func init() {
	Register(ModuleInfo{
		Name:        "Filter",
		Description: "Media filters (keyword→media) and word-ban filter",
		Commands: []CommandInfo{
			{Pattern: "filter", Handler: filterHandler, Sudo: true},
			{Pattern: "filterban", Handler: filterBanHandler, Sudo: true},
		},
	})
}
