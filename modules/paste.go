package modules

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// PBX Elite — Paste Module
// .paste <text>  — upload text to ix.io and return link
// Also works replying to a text message

func pasteText(text string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("empty content")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// ix.io: POST form field "f:1" = content
	form := url.Values{}
	form.Set("f:1", text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://ix.io", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "PBX-Elite/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Fallback: try termbin
		return pasteTermbin(text)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	link := strings.TrimSpace(string(body))

	if link == "" || !strings.HasPrefix(link, "http") {
		return pasteTermbin(text)
	}
	return link, nil
}

// pasteTermbin — fallback paste service (plain TCP, works everywhere)
func pasteTermbin(text string) (string, error) {
	// Use hastebin public API as second fallback
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://hastebin.com/documents",
		strings.NewReader(text))
	if err != nil {
		return "", fmt.Errorf("paste failed: network error")
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", "PBX-Elite/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("all paste services unavailable — please try again later")
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// hastebin returns {"key":"abc123"}
	key := ""
	s := string(body)
	if idx := strings.Index(s, `"key":"`); idx >= 0 {
		s = s[idx+7:]
		if end := strings.Index(s, `"`); end >= 0 {
			key = s[:end]
		}
	}
	if key == "" {
		return "", fmt.Errorf("paste failed — try again")
	}
	return "https://hastebin.com/" + key, nil
}

func pasteHandler(v *events.Message, args []string) error {
	text := ""

	// Check for quoted text first
	if IsReply(v) {
		qm := GetQuotedMessage(v)
		if qm != nil {
			if qm.GetConversation() != "" {
				text = qm.GetConversation()
			} else if qm.GetExtendedTextMessage() != nil {
				text = qm.GetExtendedTextMessage().GetText()
			}
		}
	}

	// Fall back to args
	if text == "" && len(args) > 0 {
		text = strings.Join(args, " ")
	}

	if strings.TrimSpace(text) == "" {
		return ReplyErr(v,
			"📋 Usage:\n"+
				"• `.paste <text>` — paste inline text\n"+
				"• Reply to a message + `.paste` — paste quoted text")
	}

	link, err := pasteText(text)
	if err != nil {
		return ReplyErr(v, "❌ "+err.Error())
	}

	return ReplyErr(v, fmt.Sprintf("📋 *Pasted!*\n🔗 %s", link))
}

func init() {
	Register(ModuleInfo{
		Name:        "Paste",
		Description: "Paste text online and get a shareable link",
		Commands: []CommandInfo{
			{Pattern: "paste", Handler: pasteHandler, Sudo: true},
		},
	})
}
