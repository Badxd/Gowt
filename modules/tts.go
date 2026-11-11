package modules

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"PbxGb/client"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

// Uses Google TTS (free, no API key needed for short text)
func fetchGoogleTTS(text, lang string) ([]byte, error) {
	encoded := url.QueryEscape(text)
	ttsURL := fmt.Sprintf(
		"https://translate.google.com/translate_tts?ie=UTF-8&q=%s&tl=%s&client=tw-ob",
		encoded, lang,
	)

	req, err := http.NewRequest("GET", ttsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; PBX/1.0)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("TTS returned status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func ttsHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .tts <text>\nOptional lang prefix: .tts hi:<hindi text>")
	}

	fullText := strings.Join(args, " ")
	lang := "en"
	text := fullText

	if idx := strings.Index(fullText, ":"); idx > 0 && idx <= 5 {
		lang = fullText[:idx]
		text = fullText[idx+1:]
	}

	if len(text) > 200 {
		text = text[:200]
	}

	audioData, err := fetchGoogleTTS(text, lang)
	if err != nil {
		return ReplyErr(v, "❌ TTS failed: "+err.Error())
	}

	wac := client.Get()
	chat := v.Info.Chat

	// Upload as audio document
	resp, err := wac.Upload(context.Background(), audioData, 0) // 0 = Document
	if err != nil {
		return ReplyErr(v, "❌ Upload failed: "+err.Error())
	}

	caption := fmt.Sprintf("🔊 TTS: _%s_", text)
	mimetype := "audio/mpeg"
	filename := "tts.mp3"

	_, err = wac.SendMessage(context.Background(), chat, &waProto.Message{
		DocumentMessage: &waProto.DocumentMessage{
			URL:           &resp.URL,
			DirectPath:    &resp.DirectPath,
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    &resp.FileLength,
			Mimetype:      &mimetype,
			FileName:      &filename,
			Caption:       &caption,
			ContextInfo: &waProto.ContextInfo{
				StanzaID:      &v.Info.ID,
				Participant:   strPtr(v.Info.Sender.String()),
				QuotedMessage: v.Message,
			},
		},
	})
	if err != nil {
		return ReplyErr(v, "❌ Send failed: "+err.Error())
	}
	return nil
}

func init() {
	Register(ModuleInfo{
		Name:        "TTS",
		Description: "Text-to-speech via Google TTS",
		Commands: []CommandInfo{
			{Pattern: "tts", Handler: ttsHandler, Sudo: true},
		},
	})
}
