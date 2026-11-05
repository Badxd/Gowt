package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

func googleTranslate(text, targetLang string) (string, error) {
	encoded := url.QueryEscape(text)
	apiURL := fmt.Sprintf(
		"https://translate.googleapis.com/translate_a/single?client=gtx&sl=auto&tl=%s&dt=t&q=%s",
		targetLang, encoded,
	)

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Response is a nested array: [[[translated, original, ...], ...], ...]
	var raw []interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", fmt.Errorf("parse error")
	}

	var result strings.Builder
	if len(raw) > 0 {
		if parts, ok := raw[0].([]interface{}); ok {
			for _, part := range parts {
				if seg, ok := part.([]interface{}); ok && len(seg) > 0 {
					if s, ok := seg[0].(string); ok {
						result.WriteString(s)
					}
				}
			}
		}
	}

	if result.Len() == 0 {
		return "", fmt.Errorf("empty translation")
	}
	return result.String(), nil
}

// .tr <lang> <text or reply>
// Example: .tr hi Hello world
func translateHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .tr <language_code> [text]\nExample: .tr hi Hello world\nOr reply to a message with .tr hi")
	}

	targetLang := args[0]
	text := ""

	if len(args) > 1 {
		text = strings.Join(args[1:], " ")
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
		return ReplyErr(v, "❌ No text to translate. Provide text or reply to a message.")
	}

	translated, err := googleTranslate(text, targetLang)
	if err != nil {
		return ReplyErr(v, "❌ Translation failed: "+err.Error())
	}

	return ReplyErr(v, fmt.Sprintf("🌐 *Translated to %s:*\n\n%s", targetLang, translated))
}

func init() {
	Register(ModuleInfo{
		Name:        "Translate",
		Description: "Translate text using Google Translate",
		Commands: []CommandInfo{
			{Pattern: "tr", Handler: translateHandler, Sudo: true},
		},
	})
}
