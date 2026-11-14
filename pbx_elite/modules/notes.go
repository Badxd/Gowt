package modules

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// Per-chat note storage (persistent in-memory, reset on restart)
// .save <name> <text>    — save a note
// .note <name>           — get a saved note
// .notes                 — list all notes in this chat
// .delnote <name>        — delete a note
// .clearnotes            — delete all notes in this chat

type noteEntry struct {
	Name    string
	Content string
	SavedBy string
	SavedAt time.Time
}

var (
	notesMu  sync.RWMutex
	notesMap = make(map[string]map[string]*noteEntry) // chatJID -> noteName -> entry
)

func saveNoteHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		return ReplyErr(v, "Usage: .save <name> <content>\nExample: .save rules No spamming allowed!")
	}

	name := strings.ToLower(args[0])
	content := strings.Join(args[1:], " ")
	chat := v.Info.Chat.String()

	notesMu.Lock()
	if notesMap[chat] == nil {
		notesMap[chat] = make(map[string]*noteEntry)
	}
	notesMap[chat][name] = &noteEntry{
		Name:    name,
		Content: content,
		SavedBy: v.Info.Sender.User,
		SavedAt: time.Now(),
	}
	notesMu.Unlock()

	return ReplyErr(v, fmt.Sprintf("✅ Note *%s* saved.", name))
}

func getNoteHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .note <name>")
	}

	name := strings.ToLower(args[0])
	chat := v.Info.Chat.String()

	notesMu.RLock()
	chatNotes, ok := notesMap[chat]
	var entry *noteEntry
	if ok {
		entry = chatNotes[name]
	}
	notesMu.RUnlock()

	if entry == nil {
		return ReplyErr(v, fmt.Sprintf("❌ No note found with name *%s*.", name))
	}

	return ReplyErr(v, fmt.Sprintf("📝 *%s*\n\n%s", entry.Name, entry.Content))
}

func listNotesHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()

	notesMu.RLock()
	chatNotes := notesMap[chat]
	notesMu.RUnlock()

	if len(chatNotes) == 0 {
		return ReplyErr(v, "ℹ️ No notes saved in this chat. Use .save <name> <text> to add one.")
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("📝 *Saved Notes (%d):*", len(chatNotes)), "")
	i := 1
	for name := range chatNotes {
		lines = append(lines, fmt.Sprintf("%d. %s", i, name))
		i++
	}
	lines = append(lines, "", "Use *.note <name>* to retrieve.")
	return ReplyErr(v, strings.Join(lines, "\n"))
}

func delNoteHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .delnote <name>")
	}

	name := strings.ToLower(args[0])
	chat := v.Info.Chat.String()

	notesMu.Lock()
	chatNotes := notesMap[chat]
	_, existed := chatNotes[name]
	if existed {
		delete(chatNotes, name)
	}
	notesMu.Unlock()

	if !existed {
		return ReplyErr(v, fmt.Sprintf("❌ No note found with name *%s*.", name))
	}
	return ReplyErr(v, fmt.Sprintf("✅ Note *%s* deleted.", name))
}

func clearNotesHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()

	notesMu.Lock()
	count := len(notesMap[chat])
	delete(notesMap, chat)
	notesMu.Unlock()

	if count == 0 {
		return ReplyErr(v, "ℹ️ No notes to clear.")
	}
	return ReplyErr(v, fmt.Sprintf("✅ Cleared *%d* note(s).", count))
}

func init() {
	Register(ModuleInfo{
		Name:        "Notes",
		Description: "Save and retrieve per-chat notes",
		Commands: []CommandInfo{
			{Pattern: "addnote", Handler: saveNoteHandler, Sudo: true},
			{Pattern: "note", Handler: getNoteHandler, Sudo: false},
			{Pattern: "notes", Handler: listNotesHandler, Sudo: false},
			{Pattern: "delnote", Handler: delNoteHandler, Sudo: true},
			{Pattern: "clearnotes", Handler: clearNotesHandler, Sudo: true},
		},
	})
}
