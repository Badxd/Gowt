package modules

import (
	"fmt"
	"strings"
	"sync"

	"PbxGb/database"

	"go.mau.fi/whatsmeow/types/events"
)

// .ar add <trigger> | <reply>   — add auto-reply rule
// .ar del <trigger>             — remove auto-reply rule
// .ar list                      — list all rules
// .ar clear                     — remove all rules
// .ar on / .ar off              — enable/disable auto-reply globally

type arRule struct {
	Trigger string
	Reply   string
}

var (
	arMu      sync.RWMutex
	arRules   []arRule
	arEnabled = true
)

// ProcessAutoReply is exported and wired into HandleMessage
func ProcessAutoReply(v *events.Message) {
	if v.Info.IsFromMe {
		return
	}

	arMu.RLock()
	enabled := arEnabled
	rules := make([]arRule, len(arRules))
	copy(rules, arRules)
	arMu.RUnlock()

	if !enabled || len(rules) == 0 {
		return
	}

	// extract text
	text := ""
	if t := v.Message.GetConversation(); t != "" {
		text = t
	} else if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		text = ext.GetText()
	}
	if text == "" {
		return
	}

	textLower := strings.ToLower(strings.TrimSpace(text))

	for _, rule := range rules {
		trigger := strings.ToLower(rule.Trigger)
		if textLower == trigger || strings.Contains(textLower, trigger) {
			ReplyErr(v, rule.Reply) //nolint:errcheck
			return                  // first match wins
		}
	}
}

func arHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .ar add <trigger> | <reply>\n.ar del <trigger>\n.ar list\n.ar clear\n.ar on/off")
	}

	switch strings.ToLower(args[0]) {
	case "add":
		rest := strings.Join(args[1:], " ")
		parts := strings.SplitN(rest, "|", 2)
		if len(parts) != 2 {
			return ReplyErr(v, "Usage: .ar add <trigger> | <reply>\nUse `|` to separate trigger from reply.")
		}
		trigger := strings.TrimSpace(parts[0])
		reply := strings.TrimSpace(parts[1])

		if trigger == "" || reply == "" {
			return ReplyErr(v, "❌ Trigger and reply cannot be empty.")
		}

		arMu.Lock()
		// check duplicate
		for _, r := range arRules {
			if strings.ToLower(r.Trigger) == strings.ToLower(trigger) {
				arMu.Unlock()
				return ReplyErr(v, fmt.Sprintf("⚠️ Trigger `%s` already exists. Delete it first with `.ar del %s`.", trigger, trigger))
			}
		}
		arRules = append(arRules, arRule{Trigger: trigger, Reply: reply})
		count := len(arRules)
		arMu.Unlock()

		_ = database.SaveAutoReplyRule(trigger, reply)
		return ReplyErr(v, fmt.Sprintf("✅ Auto-reply added. Total rules: %d\n*Trigger:* `%s`\n*Reply:* %s", count, trigger, reply))

	case "del", "delete", "remove":
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .ar del <trigger>")
		}
		trigger := strings.ToLower(strings.Join(args[1:], " "))

		arMu.Lock()
		before := len(arRules)
		var updated []arRule
		for _, r := range arRules {
			if strings.ToLower(r.Trigger) != trigger {
				updated = append(updated, r)
			}
		}
		arRules = updated
		after := len(arRules)
		arMu.Unlock()

		if before == after {
			return ReplyErr(v, fmt.Sprintf("❌ No rule found for trigger: `%s`", trigger))
		}
		_ = database.DeleteAutoReplyRule(trigger)
		return ReplyErr(v, fmt.Sprintf("🗑️ Removed auto-reply for: `%s`", trigger))

	case "list":
		arMu.RLock()
		rules := make([]arRule, len(arRules))
		copy(rules, arRules)
		enabled := arEnabled
		arMu.RUnlock()

		if len(rules) == 0 {
			return ReplyErr(v, "📋 No auto-reply rules set.")
		}

		status := "🟢 enabled"
		if !enabled {
			status = "🔴 disabled"
		}

		var lines []string
		lines = append(lines, fmt.Sprintf("📋 *Auto-Reply Rules (%d)* — %s", len(rules), status))
		lines = append(lines, "")
		for i, r := range rules {
			lines = append(lines, fmt.Sprintf("%d. `%s` → %s", i+1, r.Trigger, r.Reply))
		}
		return ReplyErr(v, strings.Join(lines, "\n"))

	case "clear":
		arMu.Lock()
		count := len(arRules)
		arRules = nil
		arMu.Unlock()
		_ = database.ClearAutoReplyRules()
		return ReplyErr(v, fmt.Sprintf("🗑️ Cleared %d auto-reply rules.", count))

	case "on":
		arMu.Lock()
		arEnabled = true
		arMu.Unlock()
		_ = database.SaveAutoReplyEnabled(true)
		return ReplyErr(v, "✅ Auto-reply enabled.")

	case "off":
		arMu.Lock()
		arEnabled = false
		arMu.Unlock()
		_ = database.SaveAutoReplyEnabled(false)
		return ReplyErr(v, "❌ Auto-reply disabled.")

	default:
		return ReplyErr(v, "Usage: .ar add/del/list/clear/on/off")
	}
}

func init() {
	// Restore persisted rules and enabled state from MongoDB.
	loaded := database.LoadAutoReplyRules()
	arMu.Lock()
	for _, r := range loaded {
		arRules = append(arRules, arRule{Trigger: r.Trigger, Reply: r.Reply})
	}
	arEnabled = database.GetAutoReplySettings().Enabled
	arMu.Unlock()

	Register(ModuleInfo{
		Name:        "Auto Reply",
		Description: "Set keyword-triggered automatic replies",
		Commands: []CommandInfo{
			{Pattern: "ar", Handler: arHandler, Sudo: true},
			{Pattern: "autoreply", Handler: arHandler, Sudo: true},
		},
	})
}
