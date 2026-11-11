package modules

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"PbxGb/client"
	"PbxGb/database"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// ── In-memory cache (loaded from DB at startup) ───────────────────────────────

var (
	pmMu      sync.RWMutex
	pmEnabled  bool
	pmMaxWarns int
	pmCustomMsg string
)

// InitPMCache loads PM settings from the database into memory.
// Must be called from main() after database.Connect() succeeds.
func InitPMCache() {
	s := database.GetPMSettings()
	pmMu.Lock()
	pmEnabled   = s.Enabled
	pmMaxWarns  = s.MaxWarns
	if pmMaxWarns < 1 {
		pmMaxWarns = 3
	}
	pmCustomMsg = s.CustomMsg
	pmMu.Unlock()
}

// ── Block helper ─────────────────────────────────────────────────────────────

const (
	blocklistAdd    = events.BlocklistChangeAction("block")
	blocklistRemove = events.BlocklistChangeAction("unblock")
)

func blockJID(jid types.JID, block bool) {
	wac := client.Get()
	action := blocklistAdd
	if !block {
		action = blocklistRemove
	}
	ctx := context.Background()
	if _, err := wac.UpdateBlocklist(ctx, jid, action); err != nil {
		// Retry once on failure.
		_, _ = wac.UpdateBlocklist(ctx, jid, action)
	}
}

// ── HandlePM — called on every incoming message ───────────────────────────────

func HandlePM(v *events.Message) {
	// Only process private DMs (not groups).
	if v.Info.Chat.Server != types.DefaultUserServer &&
		v.Info.Chat.Server != types.HiddenUserServer {
		return
	}

	// Our own messages: auto-approve the chat (classic PM-permit), never warn.
	if v.Info.IsFromMe {
		autoApproveOnOutgoing(v)
		return
	}

	// Ignore backlog after reconnects, reactions and protocol messages.
	if isStale(v) || !isRealMessage(v) {
		return
	}

	wac := client.Get()
	if wac.Store.ID != nil && v.Info.Sender.User == wac.Store.ID.User {
		return
	}

	pmMu.RLock()
	enabled := pmEnabled
	pmMu.RUnlock()
	if !enabled {
		return
	}

	sender := senderKey(v)

	// Owner and sudo users are always allowed.
	if isOwner(v) || isSudoSender(v) {
		return
	}

	// Fetch this sender's record from DB.
	rec := database.GetPMRecord(sender)

	// Approved users pass through freely.
	if rec.Approved {
		return
	}

	// Blocked users are silently ignored.
	if rec.Blocked {
		return
	}

	// Increment warn count and persist.
	rec.WarnCount++

	pmMu.RLock()
	maxW    := pmMaxWarns
	custom := pmCustomMsg
	pmMu.RUnlock()

	remaining := maxW - rec.WarnCount
	if remaining < 0 {
		remaining = 0
	}

	senderJID := v.Info.Chat

	// Auto-block when warn limit is reached.
	if rec.WarnCount >= maxW {
		rec.Blocked   = true
		rec.WarnCount = 0
		_ = database.UpsertPMRecord(sender, false, true, 0)

		blockJID(senderJID, true)

		blockMsg := fmt.Sprintf(
			"╔══════════════════════╗\n"+
				"║  *PBX Elite PM Permit* 🚫  ║\n"+
				"╚══════════════════════╝\n\n"+
				"☠️ *You have been BLOCKED!*\n"+
				"Reason: Spamming owner's DM.\n\n"+
				"Contact the owner on another platform.",
		)
		_, _ = wac.SendMessage(context.Background(), senderJID, &waProto.Message{
			Conversation: &blockMsg,
		})
		return
	}

	// Save updated warn count.
	_ = database.UpsertPMRecord(sender, false, false, rec.WarnCount)

	// Send warning message.
	var finalMsg string
	if custom != "" {
		finalMsg = fmt.Sprintf(
			"╔══════════════════════╗\n"+
				"║  *PBX Elite PM Permit* 👻  ║\n"+
				"╚══════════════════════╝\n\n"+
				"%s\n\n"+
				"⚠️ *Warning %d/%d* — %d remaining before auto-block!",
			custom, rec.WarnCount, maxW, remaining,
		)
	} else {
		finalMsg = fmt.Sprintf(
			"╔══════════════════════╗\n"+
				"║  *PBX Elite PM Permit* 👻  ║\n"+
				"╚══════════════════════╝\n\n"+
				"👋 *Hey %s!*\n"+
				"❤️ The owner is currently offline — please do not spam!\n"+
				"⚡ Spamming will result in an automatic block.\n\n"+
				"⚠️ *Warning %d/%d* — %d more warning(s) before block! ☠️",
			v.Info.PushName, rec.WarnCount, maxW, remaining,
		)
	}

	_, _ = wac.SendMessage(context.Background(), senderJID, &waProto.Message{
		Conversation: &finalMsg,
	})
}

// pmTarget resolves reply / number argument / current DM chat to one canonical key.
func pmTarget(v *events.Message, args []string) string {
	t := resolveTarget(v, args)
	if t == "" && IsDirectMessage(v) {
		t = v.Info.Chat.String()
	}
	if t == "" {
		return ""
	}
	return canonKey(t)
}

// ── COMMANDS ──────────────────────────────────────────────────────────────────

func pmSecToggleHandler(v *events.Message, args []string) error {
	pmMu.RLock()
	enabled := pmEnabled
	maxW    := pmMaxWarns
	pmMu.RUnlock()

	if len(args) == 0 {
		state := "🟢 ON"
		if !enabled {
			state = "🔴 OFF"
		}
		return ReplyErr(v, fmt.Sprintf("🔐 PM Permit: *%s*\nMax warnings: *%d*", state, maxW))
	}

	switch strings.ToLower(args[0]) {
	case "on":
		pmMu.Lock()
		pmEnabled = true
		pmMu.Unlock()
		_ = database.SavePMSettings(true, maxW, pmCustomMsg)
		return ReplyErr(v, "✅ PM Permit *ON*\nUnknown users will receive warnings. Auto-block activates after max warnings.")
	case "off":
		pmMu.Lock()
		pmEnabled = false
		pmMu.Unlock()
		_ = database.SavePMSettings(false, maxW, pmCustomMsg)
		return ReplyErr(v, "🔴 PM Permit *OFF*")
	}
	return ReplyErr(v, "❌ Usage: .pmsec on / off")
}

func approveHandler(v *events.Message, args []string) error {
	target := pmTarget(v, args)
	if target == "" {
		return ReplyErr(v, "❌ Reply to a user or use: .approve 9198xxxxxx")
	}

	_ = database.UpsertPMRecord(target, true, false, 0)

	user := strings.Split(target, "@")[0]
	blockJID(types.NewJID(user, types.DefaultUserServer), false)
	return ReplyErr(v, fmt.Sprintf("✅ Approved & Unblocked: *+%s*", user))
}

func disapproveHandler(v *events.Message, args []string) error {
	target := pmTarget(v, args)
	if target == "" {
		return ReplyErr(v, "❌ Reply to a user or use: .disapprove 9198xxxxxx")
	}

	rec := database.GetPMRecord(target)
	rec.Approved = false
	_ = database.UpsertPMRecord(target, false, rec.Blocked, rec.WarnCount)
	return ReplyErr(v, fmt.Sprintf("✅ Disapproved: *%s*", strings.Split(target, "@")[0]))
}

func blockHandler(v *events.Message, args []string) error {
	target := pmTarget(v, args)
	if target == "" {
		return ReplyErr(v, "❌ Reply to a user or use: .block 9198xxxxxx")
	}

	_ = database.UpsertPMRecord(target, false, true, 0)

	user := strings.Split(target, "@")[0]
	blockJID(types.NewJID(user, types.DefaultUserServer), true)
	return ReplyErr(v, fmt.Sprintf("🚫 Blocked: *+%s*", user))
}

func unblockHandler(v *events.Message, args []string) error {
	target := pmTarget(v, args)
	if target == "" {
		return ReplyErr(v, "❌ Reply to a user or use: .unblock 9198xxxxxx")
	}

	_ = database.UpsertPMRecord(target, false, false, 0)

	user := strings.Split(target, "@")[0]
	blockJID(types.NewJID(user, types.DefaultUserServer), false)
	return ReplyErr(v, fmt.Sprintf("✅ Unblocked: *+%s*", user))
}

func allowListHandler(v *events.Message, args []string) error {
	records, err := database.ListApprovedPM()
	if err != nil || len(records) == 0 {
		return ReplyErr(v, "ℹ️ No approved users yet.")
	}

	lines := []string{
		"╔══════════════════════╗",
		"║  ✅ APPROVED PM LIST  ║",
		"╚══════════════════════╝",
		"",
	}
	for i, rec := range records {
		lines = append(lines, fmt.Sprintf("%d. +%s", i+1, strings.Split(rec.JID, "@")[0]))
	}
	return ReplyErr(v, strings.Join(lines, "\n"))
}

func setPmMsgHandler(v *events.Message, args []string) error {
	msg := GetArgs(v)

	pmMu.Lock()
	pmCustomMsg = msg
	enabled    := pmEnabled
	maxW       := pmMaxWarns
	pmMu.Unlock()

	_ = database.SavePMSettings(enabled, maxW, msg)

	if msg == "" {
		return ReplyErr(v, "✅ PM Permit message reset to default.")
	}
	return ReplyErr(v, fmt.Sprintf("✅ PM Permit custom message set:\n\n%s", msg))
}

func pmWarnSetHandler(v *events.Message, args []string) error {
	pmMu.RLock()
	current  := pmMaxWarns
	enabled  := pmEnabled
	customMsg := pmCustomMsg
	pmMu.RUnlock()

	if len(args) == 0 {
		return ReplyErr(v, fmt.Sprintf("⚠️ Max warnings before auto-block: *%d*", current))
	}

	var n int
	fmt.Sscanf(args[0], "%d", &n)
	if n < 1 {
		n = 1
	}

	pmMu.Lock()
	pmMaxWarns = n
	pmMu.Unlock()

	_ = database.SavePMSettings(enabled, n, customMsg)
	return ReplyErr(v, fmt.Sprintf("✅ Max warnings set to: *%d*", n))
}

func pmResetHandler(v *events.Message, args []string) error {
	if err := database.ResetAllPMRecords(); err != nil {
		return ReplyErr(v, "❌ Failed to reset PM records. Check database connection.")
	}
	return ReplyErr(v, "✅ All PM warnings and blocks have been reset.")
}

// pmStatusHandler shows whether PM Permit is on/off (alias for pmsec with no args).
func pmStatusHandler(v *events.Message, args []string) error {
	return pmSecToggleHandler(v, nil)
}

func init() {
	Register(ModuleInfo{
		Name:        "PMSecurity",
		Description: "PM Permit — persistent DB-backed warnings, approvals, and auto-block",
		Commands: []CommandInfo{
			{Pattern: "pmsec",       Handler: pmSecToggleHandler, Sudo: true},
			{Pattern: "pmstatus",    Handler: pmStatusHandler,    Sudo: true},
			{Pattern: "approve",     Handler: approveHandler,     Sudo: true},
			{Pattern: "allow",       Handler: approveHandler,     Sudo: true},
			{Pattern: "disapprove",  Handler: disapproveHandler,  Sudo: true},
			{Pattern: "disallow",    Handler: disapproveHandler,  Sudo: true},
			{Pattern: "block",       Handler: blockHandler,       Sudo: true},
			{Pattern: "unblock",     Handler: unblockHandler,     Sudo: true},
			{Pattern: "allowlist",   Handler: allowListHandler,   Sudo: true},
			{Pattern: "approvelist", Handler: allowListHandler,   Sudo: true},
			{Pattern: "setpmmsg",    Handler: setPmMsgHandler,    Sudo: true},
			{Pattern: "pmwarn",      Handler: pmWarnSetHandler,   Sudo: true},
			{Pattern: "pmreset",     Handler: pmResetHandler,     Sudo: true},
		},
	})
}
