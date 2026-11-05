package modules

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

// paste.rs / dpaste fallback
func pasteToService(text string) (string, error) {
	// Try dpaste.com first (no auth needed)
	data := url.Values{}
	data.Set("content", text)
	data.Set("syntax", "text")
	data.Set("expiry_days", "7")

	resp, err := http.PostForm("https://dpaste.com/api/v2/", data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		body, _ := io.ReadAll(resp.Body)
		link := strings.TrimSpace(string(body))
		return link, nil
	}
	return "", fmt.Errorf("dpaste returned status %d", resp.StatusCode)
}

func pasteHandler(v *events.Message, args []string) error {
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
		return ReplyErr(v, "Usage: .paste <text> or reply to a message")
	}

	link, err := pasteToService(text)
	if err != nil {
		return ReplyErr(v, "❌ Failed to paste: "+err.Error())
	}
	return ReplyErr(v, fmt.Sprintf("📋 Pasted!\n🔗 %s", link))
}

func init() {
	Register(ModuleInfo{
		Name:        "Paste",
		Description: "Paste text to dpaste.com and get a link",
		Commands: []CommandInfo{
			{Pattern: "paste", Handler: pasteHandler, Sudo: true},
		},
	})
}
