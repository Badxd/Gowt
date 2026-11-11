package modules

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types/events"
)

// .wc    — word/char/line count of quoted or provided text
// .count — alias

func wcHandler(v *events.Message, args []string) error {
	t := getText(v, args)
	if t == "" {
		return ReplyErr(v, "Usage: .wc <text> or reply to a message")
	}

	words := len(strings.Fields(t))
	chars := utf8.RuneCountInString(t)
	lines := strings.Count(t, "\n") + 1
	bytes_ := len(t)
	sentences := strings.Count(t, ".") + strings.Count(t, "!") + strings.Count(t, "?")

	return ReplyErr(v, fmt.Sprintf(
		"📊 *Word Count*\n"+
			"📝 Words: *%d*\n"+
			"🔤 Chars: *%d*\n"+
			"📏 Lines: *%d*\n"+
			"📦 Bytes: *%d*\n"+
			"❓ Sentences: *%d*",
		words, chars, lines, bytes_, sentences,
	))
}

func init() {
	Register(ModuleInfo{
		Name:        "Word Count",
		Description: "Count words, chars, lines in text",
		Commands: []CommandInfo{
			{Pattern: "wc", Handler: wcHandler, Sudo: true},
			{Pattern: "count", Handler: wcHandler, Sudo: true},
		},
	})
}
