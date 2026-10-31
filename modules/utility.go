package modules

import (
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"io"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// ── .calc ────────────────────────────────────────────────────────────────────
func calcHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "❌ Use: .calc 2+2\nSupported: + - * / ^ %\nExample: .calc (10+5)*3")
	}
	expr := strings.Join(args, "")
	result, err := evalMath(expr)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Invalid expression: %v\nExample: .calc 10*5+3", err))
	}
	return ReplyErr(v, fmt.Sprintf("🧮 *Calculator*\n\n`%s`\n= *%s*", expr, result))
}

func evalMath(expr string) (string, error) {
	expr = strings.ReplaceAll(expr, " ", "")
	val, _, err := parseExpr(expr, 0)
	if err != nil {
		return "", err
	}
	f, _ := val.Float64()
	if val.IsInt() {
		return val.RatString(), nil
	}
	return fmt.Sprintf("%.4f", f), nil
}

func parseExpr(s string, pos int) (*big.Rat, int, error) {
	left, pos, err := parseTerm(s, pos)
	if err != nil {
		return nil, pos, err
	}
	for pos < len(s) && (s[pos] == '+' || s[pos] == '-') {
		op := s[pos]
		pos++
		right, newPos, err := parseTerm(s, pos)
		if err != nil {
			return nil, newPos, err
		}
		pos = newPos
		if op == '+' {
			left.Add(left, right)
		} else {
			left.Sub(left, right)
		}
	}
	return left, pos, nil
}

func parseTerm(s string, pos int) (*big.Rat, int, error) {
	left, pos, err := parseFactor(s, pos)
	if err != nil {
		return nil, pos, err
	}
	for pos < len(s) && (s[pos] == '*' || s[pos] == '/') {
		op := s[pos]
		pos++
		right, newPos, err := parseFactor(s, pos)
		if err != nil {
			return nil, newPos, err
		}
		pos = newPos
		if op == '*' {
			left.Mul(left, right)
		} else {
			if right.Sign() == 0 {
				return nil, pos, fmt.Errorf("division by zero")
			}
			left.Quo(left, right)
		}
	}
	return left, pos, nil
}

func parseFactor(s string, pos int) (*big.Rat, int, error) {
	if pos >= len(s) {
		return nil, pos, fmt.Errorf("unexpected end")
	}
	if s[pos] == '(' {
		pos++
		val, pos, err := parseExpr(s, pos)
		if err != nil {
			return nil, pos, err
		}
		if pos >= len(s) || s[pos] != ')' {
			return nil, pos, fmt.Errorf("missing )")
		}
		return val, pos + 1, nil
	}
	neg := false
	if s[pos] == '-' {
		neg = true
		pos++
	}
	start := pos
	for pos < len(s) && (s[pos] >= '0' && s[pos] <= '9' || s[pos] == '.') {
		pos++
	}
	if start == pos {
		return nil, pos, fmt.Errorf("expected number at pos %d", pos)
	}
	r := new(big.Rat)
	r.SetString(s[start:pos])
	if neg {
		r.Neg(r)
	}
	return r, pos, nil
}

// ── .remind ──────────────────────────────────────────────────────────────────
var remindTimers sync.Map

func remindHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		return ReplyErr(v, "❌ Use: .remind 10m Call mom\nTime formats: 10s, 5m, 2h, 1d")
	}

	timeStr := args[0]
	msg := strings.Join(args[1:], " ")

	dur, err := parseDuration(timeStr)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Invalid time: %s\nUse: 10s, 5m, 2h, 1d", timeStr))
	}

	chat := v.Info.Chat
	chatStr := chat.String()

	go func() {
		time.Sleep(dur)
		reminderMsg := fmt.Sprintf("⏰ *REMINDER*\n\n%s", msg)
		// reuse the sendText helper
		SendText(v, reminderMsg)
		remindTimers.Delete(chatStr + msg)
	}()

	return ReplyErr(v, fmt.Sprintf("⏰ Reminder set!\n*%s* baad yaad dilaunga:\n_%s_", timeStr, msg))
}

func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("too short")
	}
	unit := s[len(s)-1]
	valStr := s[:len(s)-1]
	var val int
	fmt.Sscanf(valStr, "%d", &val)
	if val <= 0 {
		return 0, fmt.Errorf("value must be positive")
	}
	switch unit {
	case 's':
		return time.Duration(val) * time.Second, nil
	case 'm':
		return time.Duration(val) * time.Minute, nil
	case 'h':
		return time.Duration(val) * time.Hour, nil
	case 'd':
		return time.Duration(val) * 24 * time.Hour, nil
	}
	return 0, fmt.Errorf("unknown unit %c (use s/m/h/d)", unit)
}

// ── .tinyurl ─────────────────────────────────────────────────────────────────
func tinyURLHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "❌ Use: .tinyurl https://example.com/long-url")
	}
	longURL := args[0]
	apiURL := fmt.Sprintf("https://tinyurl.com/api-create.php?url=%s", url.QueryEscape(longURL))

	resp, err := http.Get(apiURL)
	if err != nil {
		return ReplyErr(v, "❌ TinyURL service unavailable. Try later.")
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	short := strings.TrimSpace(string(body))
	if !strings.HasPrefix(short, "https://") {
		return ReplyErr(v, "❌ Could not shorten URL.")
	}
	return ReplyErr(v, fmt.Sprintf("🔗 *URL Shortener*\n\n*Original:* %s\n\n*Short:* %s", longURL, short))
}

// ── .translate ────────────────────────────────────────────────────────────────
func translateHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		return ReplyErr(v, "❌ Use: .translate hi Hello how are you\n\nLanguage codes:\n`hi` = Hindi\n`en` = English\n`pa` = Punjabi\n`ur` = Urdu\n`es` = Spanish\n`fr` = French\n`de` = German\n`ar` = Arabic\n`zh` = Chinese\n`ja` = Japanese")
	}
	targetLang := args[0]
	text := strings.Join(args[1:], " ")

	apiURL := fmt.Sprintf(
		"https://api.mymemory.translated.net/get?q=%s&langpair=auto|%s",
		url.QueryEscape(text), url.QueryEscape(targetLang),
	)

	resp, err := http.Get(apiURL)
	if err != nil {
		return ReplyErr(v, "❌ Translation service unavailable.")
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Simple JSON parse for translatedText
	start := strings.Index(bodyStr, `"translatedText":"`)
	if start == -1 {
		return ReplyErr(v, "❌ Translation failed.")
	}
	start += len(`"translatedText":"`)
	end := strings.Index(bodyStr[start:], `"`)
	if end == -1 {
		return ReplyErr(v, "❌ Translation failed.")
	}
	translated := bodyStr[start : start+end]

	return ReplyErr(v, fmt.Sprintf(
		"🌐 *Translator*\n\n*Original:*\n%s\n\n*Translated (%s):*\n%s",
		text, targetLang, translated,
	))
}

// ── .weather ─────────────────────────────────────────────────────────────────
func weatherHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "❌ Use: .weather Delhi\nOr: .weather New York")
	}
	city := url.QueryEscape(strings.Join(args, "+"))
	apiURL := fmt.Sprintf("https://wttr.in/%s?format=3", city)

	resp, err := http.Get(apiURL)
	if err != nil {
		return ReplyErr(v, "❌ Weather service unavailable.")
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	weather := strings.TrimSpace(string(body))

	if weather == "" || strings.Contains(weather, "Unknown") {
		return ReplyErr(v, fmt.Sprintf("❌ City not found: %s\nPlease enter a valid city name.", strings.Join(args, " ")))
	}

	// Full weather
	apiURL2 := fmt.Sprintf("https://wttr.in/%s?format=%%l:+%%C+%%t+%%h+%%w", city)
	resp2, err := http.Get(apiURL2)
	var full string
	if err == nil {
		defer resp2.Body.Close()
		b, _ := io.ReadAll(resp2.Body)
		full = strings.TrimSpace(string(b))
	}

	msg := fmt.Sprintf("🌤 *Weather Update*\n\n📍 *%s*\n\n%s", strings.Join(args, " "), weather)
	if full != "" && full != weather {
		msg += "\n\n" + full
	}
	return ReplyErr(v, msg)
}

func init() {
	Register(ModuleInfo{
		Name:        "Utility",
		Description: "calc, remind, tinyurl, translate, weather",
		Commands: []CommandInfo{
			{Pattern: "calc",      Handler: calcHandler,      Sudo: true},
			{Pattern: "remind",    Handler: remindHandler,    Sudo: true},
			{Pattern: "tinyurl",   Handler: tinyURLHandler,   Sudo: true},
			{Pattern: "translate", Handler: translateHandler, Sudo: true},
			{Pattern: "tr",        Handler: translateHandler, Sudo: true},
			{Pattern: "weather",   Handler: weatherHandler,   Sudo: true},
		},
	})
}
