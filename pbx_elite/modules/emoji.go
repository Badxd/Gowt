package modules

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
	"unicode"

	"go.mau.fi/whatsmeow/types/events"
)

// .emojify <text>  — add random fun emoji after each word
// .demojify <text> — strip all emoji from text
// .ecount <text>   — count emoji in quoted/text

var wordEmojis = []string{
	"😂", "🔥", "💯", "✨", "👀", "🎉", "💀", "🤣", "😭", "👏",
	"🙌", "🤝", "💪", "🎯", "🚀", "😎", "🤙", "👑", "❤️", "⚡",
	"🌟", "🍀", "🎊", "🥳", "🤩", "😍", "💥", "🎶", "🌈", "🏆",
}

var emojiRng = rand.New(rand.NewSource(time.Now().UnixNano()))

func emojifyHandler(v *events.Message, args []string) error {
	t := getText(v, args)
	if t == "" {
		return ReplyErr(v, "Usage: .emojify <text> or reply to a message")
	}
	words := strings.Fields(t)
	for i, w := range words {
		em := wordEmojis[emojiRng.Intn(len(wordEmojis))]
		words[i] = w + em
	}
	return ReplyErr(v, strings.Join(words, " "))
}

func demojifyHandler(v *events.Message, args []string) error {
	t := getText(v, args)
	if t == "" {
		return ReplyErr(v, "Usage: .demojify <text> or reply to a message")
	}
	var sb strings.Builder
	for _, r := range t {
		if !isEmoji(r) {
			sb.WriteRune(r)
		}
	}
	result := strings.TrimSpace(sb.String())
	if result == "" {
		return ReplyErr(v, "(no text left after removing emoji)")
	}
	return ReplyErr(v, result)
}

func ecountHandler(v *events.Message, args []string) error {
	t := getText(v, args)
	if t == "" {
		return ReplyErr(v, "Usage: .ecount <text> or reply to a message")
	}
	count := 0
	unique := make(map[rune]int)
	for _, r := range t {
		if isEmoji(r) {
			count++
			unique[r]++
		}
	}
	if count == 0 {
		return ReplyErr(v, "No emoji found in that message.")
	}
	return ReplyErr(v, fmt.Sprintf("🔢 Total emoji: *%d* | Unique: *%d*", count, len(unique)))
}

// isEmoji — rough check: ranges that cover most emoji
func isEmoji(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) && !unicode.IsPunct(r) &&
		(r >= 0x1F300 || r >= 0x2600 && r <= 0x27BF)
}

func init() {
	Register(ModuleInfo{
		Name:        "Emoji Tools",
		Description: "Add, remove, or count emoji in text",
		Commands: []CommandInfo{
			{Pattern: "emojify", Handler: emojifyHandler, Sudo: true},
			{Pattern: "demojify", Handler: demojifyHandler, Sudo: true},
			{Pattern: "ecount", Handler: ecountHandler, Sudo: true},
		},
	})
}
