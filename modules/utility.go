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


func parseDurationUtil(s string) (time.Duration, error) {
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



func init() {
	Register(ModuleInfo{
		Name:        "Utility",
		Description: "calc, remind, tinyurl, translate, weather",
		Commands: []CommandInfo{
			{Pattern: "tinyurl",   Handler: tinyURLHandler,   Sudo: true},
		},
	})
}
