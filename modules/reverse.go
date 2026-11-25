package modules

import (
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

// .reverse <text> or reply
func reverseHandler(v *events.Message, args []string) error {
	text := ""
	if len(args) > 0 {
		text = strings.Join(args, " ")
	} else if IsReply(v) {
		q := GetQuotedMessage(v)
		if q != nil {
			if q.Conversation != nil {
				text = *q.Conversation
			} else if q.ExtendedTextMessage != nil && q.ExtendedTextMessage.Text != nil {
				text = *q.ExtendedTextMessage.Text
			}
		}
	}
	if text == "" {
		return ReplyErr(v, "Usage: .reverse <text> or reply to a message")
	}
	runes := []rune(text)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return ReplyErr(v, string(runes))
}

// .upper <text>
func upperHandler(v *events.Message, args []string) error {
	text := ""
	if len(args) > 0 {
		text = strings.Join(args, " ")
	} else if IsReply(v) {
		q := GetQuotedMessage(v)
		if q != nil && q.Conversation != nil {
			text = *q.Conversation
		}
	}
	if text == "" {
		return ReplyErr(v, "Usage: .upper <text>")
	}
	return ReplyErr(v, strings.ToUpper(text))
}

// .lower <text>
func lowerHandler(v *events.Message, args []string) error {
	text := ""
	if len(args) > 0 {
		text = strings.Join(args, " ")
	} else if IsReply(v) {
		q := GetQuotedMessage(v)
		if q != nil && q.Conversation != nil {
			text = *q.Conversation
		}
	}
	if text == "" {
		return ReplyErr(v, "Usage: .lower <text>")
	}
	return ReplyErr(v, strings.ToLower(text))
}

// .mock <text> — SpOnGeBob mocking
func mockHandler(v *events.Message, args []string) error {
	text := ""
	if len(args) > 0 {
		text = strings.Join(args, " ")
	} else if IsReply(v) {
		q := GetQuotedMessage(v)
		if q != nil && q.Conversation != nil {
			text = *q.Conversation
		}
	}
	if text == "" {
		return ReplyErr(v, "Usage: .mock <text>")
	}
	runes := []rune(text)
	upper := false
	for i, ch := range runes {
		if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' {
			if upper {
				runes[i] = []rune(strings.ToUpper(string(ch)))[0]
			} else {
				runes[i] = []rune(strings.ToLower(string(ch)))[0]
			}
			upper = !upper
		}
	}
	return ReplyErr(v, string(runes))
}

func init() {
	Register(ModuleInfo{
		Name:        "TextTools",
		Description: "Text manipulation: reverse, upper, lower, mock",
		Commands: []CommandInfo{
			{Pattern: "reverse", Handler: reverseHandler, Sudo: true},
			{Pattern: "upper", Handler: upperHandler, Sudo: true},
			{Pattern: "lower", Handler: lowerHandler, Sudo: true},
			{Pattern: "mock", Handler: mockHandler, Sudo: true},
		},
	})
}
