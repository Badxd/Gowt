package modules

import (
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

var morseTable = map[rune]string{
	'A': ".-", 'B': "-...", 'C': "-.-.", 'D': "-..", 'E': ".",
	'F': "..-.", 'G': "--.", 'H': "....", 'I': "..", 'J': ".---",
	'K': "-.-", 'L': ".-..", 'M': "--", 'N': "-.", 'O': "---",
	'P': ".--.", 'Q': "--.-", 'R': ".-.", 'S': "...", 'T': "-",
	'U': "..-", 'V': "...-", 'W': ".--", 'X': "-..-", 'Y': "-.--",
	'Z': "--..",
	'0': "-----", '1': ".----", '2': "..---", '3': "...--", '4': "....-",
	'5': ".....", '6': "-....", '7': "--...", '8': "---..", '9': "----.",
	' ': "/",
}

var reverseMorse map[string]rune

func init() {
	reverseMorse = make(map[string]rune)
	for k, v := range morseTable {
		reverseMorse[v] = k
	}
}

func toMorse(text string) string {
	var parts []string
	for _, ch := range strings.ToUpper(text) {
		if code, ok := morseTable[ch]; ok {
			parts = append(parts, code)
		} else {
			parts = append(parts, "?")
		}
	}
	return strings.Join(parts, " ")
}

func fromMorse(code string) string {
	var sb strings.Builder
	words := strings.Split(code, " / ")
	for i, word := range words {
		if i > 0 {
			sb.WriteRune(' ')
		}
		letters := strings.Split(word, " ")
		for _, letter := range letters {
			if ch, ok := reverseMorse[letter]; ok {
				sb.WriteRune(ch)
			} else if letter != "" {
				sb.WriteRune('?')
			}
		}
	}
	return sb.String()
}

// .morse <text>
func morseEncodeHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .morse <text>")
	}
	text := strings.Join(args, " ")
	return ReplyErr(v, "📡 "+toMorse(text))
}

// .unmorse <morse code>
func morseDecodeHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .unmorse <morse code>")
	}
	code := strings.Join(args, " ")
	return ReplyErr(v, "📖 "+fromMorse(code))
}

func init() {
	Register(ModuleInfo{
		Name:        "Morse",
		Description: "Encode/decode Morse code",
		Commands: []CommandInfo{
			{Pattern: "morse", Handler: morseEncodeHandler, Sudo: true},
			{Pattern: "unmorse", Handler: morseDecodeHandler, Sudo: true},
		},
	})
}
