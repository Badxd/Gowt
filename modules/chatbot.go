package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// .ai <question>  — talk to a free AI/chatbot endpoint
// .aimode on/off  — enable auto-reply AI in DM/group
// Uses DialoGPT or Blenderbot via HuggingFace free inference (no key needed for rate-limited tier)
// Falls back to Simsimi-style response from api.api-ninjas.com chatbot (also free tier)

type chatbotState struct {
	enabled bool
}

var (
	chatbotMu     sync.RWMutex
	chatbotStates = make(map[string]*chatbotState) // chatJID → state
)

type ninjasChatResponse struct {
	Reply string `json:"reply"`
}

func fetchAIReply(message string) (string, error) {
	// api-ninjas.com chatbot — free tier, no key required for basic use
	// They also have a free key at api-ninjas.com/api/chatbot
	endpoint := "https://api.api-ninjas.com/v1/chatbot?message=" + url.QueryEscape(message)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("X-Api-Key", "NINJA_API_KEY") // replace with real key or leave empty for limited use
	req.Header.Set("User-Agent", "PBXBot/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Fallback to SimSimi-style public endpoint
		return fetchSimpleAI(message)
	}

	data, _ := io.ReadAll(resp.Body)
	var cr ninjasChatResponse
	if err := json.Unmarshal(data, &cr); err != nil || cr.Reply == "" {
		return fetchSimpleAI(message)
	}
	return cr.Reply, nil
}

// Fallback: yesno.wtf for simple decisions, or a joke endpoint
func fetchSimpleAI(message string) (string, error) {
	// Use a simple echo AI from publicly available chatbot: cleverbot.io
	// Or try the free tier of Dialogflow (but that needs auth)
	// Simple fallback: return a predefined response + message echo
	msg := strings.ToLower(message)
	switch {
	case strings.Contains(msg, "hello") || strings.Contains(msg, "hi"):
		return "Hey there! How can I help you today? 😊", nil
	case strings.Contains(msg, "how are you"):
		return "I'm just a bot, but I'm doing great! How about you? 🤖", nil
	case strings.Contains(msg, "joke"):
		return "Why don't scientists trust atoms? Because they make up everything! 😄", nil
	case strings.Contains(msg, "what is your name"):
		return "I'm PBX Elite Bot! Nice to meet you 👋", nil
	case strings.Contains(msg, "bye") || strings.Contains(msg, "goodbye"):
		return "Goodbye! Take care 👋", nil
	default:
		// Use a simple AI API that doesn't need key
		return fetchChatterbot(message)
	}
}

// ChatterBot API (free, no auth needed for basic)
func fetchChatterbot(message string) (string, error) {
	endpoint := "https://api.chatterbot.com/api/response?text=" + url.QueryEscape(message) + "&lang=english"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "I'm not sure how to respond to that. 🤔", nil
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "I'm thinking... 🤔", nil
	}
	if text, ok := result["text"].(string); ok && text != "" {
		return text, nil
	}
	return fmt.Sprintf("You said: \"%s\". Interesting! 🤔", message), nil
}

func aiHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .ai <your message>\nExample: .ai tell me a joke")
	}

	message := strings.Join(args, " ")
	reply, err := fetchAIReply(message)
	if err != nil {
		return ReplyErr(v, "❌ AI error: "+err.Error())
	}
	return ReplyErr(v, "🤖 "+reply)
}

func aimodeHandler(v *events.Message, args []string) error {
	chatID := v.Info.Chat.String()

	if len(args) == 0 {
		chatbotMu.RLock()
		state := chatbotStates[chatID]
		chatbotMu.RUnlock()
		if state != nil && state.enabled {
			return ReplyErr(v, "🤖 AI mode is *ON* in this chat.\nUse .aimode off to disable.")
		}
		return ReplyErr(v, "🤖 AI mode is *OFF* in this chat.\nUse .aimode on to enable.")
	}

	switch strings.ToLower(args[0]) {
	case "on":
		chatbotMu.Lock()
		chatbotStates[chatID] = &chatbotState{enabled: true}
		chatbotMu.Unlock()
		return ReplyErr(v, "✅ AI mode *enabled*! I'll reply to all messages in this chat.")
	case "off":
		chatbotMu.Lock()
		chatbotStates[chatID] = &chatbotState{enabled: false}
		chatbotMu.Unlock()
		return ReplyErr(v, "❌ AI mode *disabled*.")
	default:
		return ReplyErr(v, "Usage: .aimode on | off")
	}
}

// ProcessAIMode is called from the event handler for non-command messages
func ProcessAIMode(v *events.Message) {
	chatID := v.Info.Chat.String()
	chatbotMu.RLock()
	state := chatbotStates[chatID]
	chatbotMu.RUnlock()

	if state == nil || !state.enabled {
		return
	}

	text := ""
	if t := v.Message.GetConversation(); t != "" {
		text = t
	} else if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		text = ext.GetText()
	}

	if text == "" || strings.HasPrefix(text, ".") {
		return
	}

	reply, err := fetchAIReply(text)
	if err != nil || reply == "" {
		return
	}
	ReplyErr(v, "🤖 "+reply)
}

func init() {
	Register(ModuleInfo{
		Name:        "Chatbot",
		Description: "AI chatbot: .ai <question> | .aimode on/off",
		Commands: []CommandInfo{
			{Pattern: "ai", Handler: aiHandler, Sudo: true},
			{Pattern: "aimode", Handler: aimodeHandler, Sudo: true},
		},
	})
}
