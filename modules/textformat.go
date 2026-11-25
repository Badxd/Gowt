package modules

import (
	"strings"
	"unicode"

	"go.mau.fi/whatsmeow/types/events"
)

// Extra text formatting commands (upper/lower/mock are in reverse.go)
// .title   — Title Case
// .sponge  — SpOnGeBoB mOcK (alias of .mock with alternate toggle start)
// .clap    — add 👏 between 👏 every 👏 word
// .small   — ˢᵐᵃˡˡ superscript-like unicode
// .fancy   — 𝓯𝓪𝓷𝓬𝔂 script unicode text
// .mirror  — ɹoɹɹᴉɯ upside-down flip

var smallMap = map[rune]string{
	'a': "ᵃ", 'b': "ᵇ", 'c': "ᶜ", 'd': "ᵈ", 'e': "ᵉ",
	'f': "ᶠ", 'g': "ᵍ", 'h': "ʰ", 'i': "ⁱ", 'j': "ʲ",
	'k': "ᵏ", 'l': "ˡ", 'm': "ᵐ", 'n': "ⁿ", 'o': "ᵒ",
	'p': "ᵖ", 'q': "ᵠ", 'r': "ʳ", 's': "ˢ", 't': "ᵗ",
	'u': "ᵘ", 'v': "ᵛ", 'w': "ʷ", 'x': "ˣ", 'y': "ʸ", 'z': "ᶻ",
}

var fancyMap = map[rune]rune{
	'a': '𝓪', 'b': '𝓫', 'c': '𝓬', 'd': '𝓭', 'e': '𝓮',
	'f': '𝓯', 'g': '𝓰', 'h': '𝓱', 'i': '𝓲', 'j': '𝓳',
	'k': '𝓴', 'l': '𝓵', 'm': '𝓶', 'n': '𝓷', 'o': '𝓸',
	'p': '𝓹', 'q': '𝓺', 'r': '𝓻', 's': '𝓼', 't': '𝓽',
	'u': '𝓾', 'v': '𝓿', 'w': '𝔀', 'x': '𝔁', 'y': '𝔂', 'z': '𝔃',
	'A': '𝓐', 'B': '𝓑', 'C': '𝓒', 'D': '𝓓', 'E': '𝓔',
	'F': '𝓕', 'G': '𝓖', 'H': '𝓗', 'I': '𝓘', 'J': '𝓙',
	'K': '𝓚', 'L': '𝓛', 'M': '𝓜', 'N': '𝓝', 'O': '𝓞',
	'P': '𝓟', 'Q': '𝓠', 'R': '𝓡', 'S': '𝓢', 'T': '𝓣',
	'U': '𝓤', 'V': '𝓥', 'W': '𝓦', 'X': '𝓧', 'Y': '𝓨', 'Z': '𝓩',
}

var mirrorMap = map[rune]rune{
	'a': 'ɐ', 'b': 'q', 'c': 'ɔ', 'd': 'p', 'e': 'ǝ',
	'f': 'ɟ', 'g': 'ƃ', 'h': 'ɥ', 'i': 'ᴉ', 'j': 'ɾ',
	'k': 'ʞ', 'l': 'l', 'm': 'ɯ', 'n': 'u', 'o': 'o',
	'p': 'd', 'q': 'b', 'r': 'ɹ', 's': 's', 't': 'ʇ',
	'u': 'n', 'v': 'ʌ', 'w': 'ʍ', 'x': 'x', 'y': 'ʎ', 'z': 'z',
	'A': '∀', 'B': 'B', 'C': 'Ɔ', 'D': 'D', 'E': 'Ǝ',
	'F': 'Ⅎ', 'G': 'פ', 'H': 'H', 'I': 'I', 'J': 'ſ',
	'K': 'K', 'L': '˥', 'M': 'W', 'N': 'N', 'O': 'O',
	'P': 'Ԁ', 'Q': 'Q', 'R': 'R', 'S': 'S', 'T': '⊥',
	'U': '∩', 'V': 'Λ', 'W': 'M', 'X': 'X', 'Y': '⅄', 'Z': 'Z',
}

func getText(v *events.Message, args []string) string {
	if len(args) > 0 {
		return strings.Join(args, " ")
	}
	if v.Message.GetExtendedTextMessage() != nil {
		ctx := v.Message.GetExtendedTextMessage().GetContextInfo()
		if ctx != nil && ctx.GetQuotedMessage() != nil {
			q := ctx.GetQuotedMessage()
			if q.GetConversation() != "" {
				return q.GetConversation()
			}
			if q.GetExtendedTextMessage() != nil {
				return q.GetExtendedTextMessage().GetText()
			}
		}
	}
	return ""
}

func titleHandler(v *events.Message, args []string) error {
	t := getText(v, args)
	if t == "" {
		return ReplyErr(v, "Usage: .title <text> or reply to a message")
	}
	return ReplyErr(v, strings.Title(t)) //nolint:staticcheck
}

func spongeHandler(v *events.Message, args []string) error {
	t := getText(v, args)
	if t == "" {
		return ReplyErr(v, "Usage: .sponge <text> or reply to a message")
	}
	runes := []rune(t)
	upper := true
	for i, r := range runes {
		if unicode.IsLetter(r) {
			if upper {
				runes[i] = unicode.ToUpper(r)
			} else {
				runes[i] = unicode.ToLower(r)
			}
			upper = !upper
		}
	}
	return ReplyErr(v, string(runes))
}

func clapHandler(v *events.Message, args []string) error {
	t := getText(v, args)
	if t == "" {
		return ReplyErr(v, "Usage: .clap <text> or reply to a message")
	}
	words := strings.Fields(t)
	return ReplyErr(v, strings.Join(words, " 👏 "))
}

func smallHandler(v *events.Message, args []string) error {
	t := getText(v, args)
	if t == "" {
		return ReplyErr(v, "Usage: .small <text> or reply to a message")
	}
	var sb strings.Builder
	for _, r := range strings.ToLower(t) {
		if s, ok := smallMap[r]; ok {
			sb.WriteString(s)
		} else {
			sb.WriteRune(r)
		}
	}
	return ReplyErr(v, sb.String())
}

func fancyHandler(v *events.Message, args []string) error {
	t := getText(v, args)
	if t == "" {
		return ReplyErr(v, "Usage: .fancy <text> or reply to a message")
	}
	var sb strings.Builder
	for _, r := range t {
		if fr, ok := fancyMap[r]; ok {
			sb.WriteRune(fr)
		} else {
			sb.WriteRune(r)
		}
	}
	return ReplyErr(v, sb.String())
}

func mirrorHandler(v *events.Message, args []string) error {
	t := getText(v, args)
	if t == "" {
		return ReplyErr(v, "Usage: .mirror <text> or reply to a message")
	}
	runes := []rune(t)
	result := make([]rune, len(runes))
	for i, r := range runes {
		if m, ok := mirrorMap[r]; ok {
			result[i] = m
		} else {
			result[i] = r
		}
	}
	for l, r := 0, len(result)-1; l < r; l, r = l+1, r-1 {
		result[l], result[r] = result[r], result[l]
	}
	return ReplyErr(v, string(result))
}

func init() {
	Register(ModuleInfo{
		Name:        "Text Format Extra",
		Description: "Extra text transforms: title, sponge, clap, small, fancy, mirror",
		Commands: []CommandInfo{
			{Pattern: "title", Handler: titleHandler, Sudo: true},
			{Pattern: "sponge", Handler: spongeHandler, Sudo: true},
			{Pattern: "clap", Handler: clapHandler, Sudo: true},
			{Pattern: "small", Handler: smallHandler, Sudo: true},
			{Pattern: "fancy", Handler: fancyHandler, Sudo: true},
			{Pattern: "mirror", Handler: mirrorHandler, Sudo: true},
		},
	})
}
