package modules

import (
	"fmt"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow/types/events"
)

// Sudo user management
// .addsudo <number|@mention>   — add a sudo user
// .delsudo <number|@mention>   — remove a sudo user
// .sudolist                    — list all sudo users

var (
	sudoMu    sync.RWMutex
	sudoUsers = make(map[string]bool) // normalized JID string -> true
)

// IsSudoUser checks if the given JID string is a sudo user or the owner.
// Called by module.go dispatch loop.
func IsSudoUser(jid string) bool {
	// strip device suffix e.g. "1234567890:5@s.whatsapp.net" -> "1234567890@s.whatsapp.net"
	normalized := normalizeJIDForSudo(jid)
	sudoMu.RLock()
	defer sudoMu.RUnlock()
	return sudoUsers[normalized]
}

// AddSudo adds a JID to the sudo list (called from settings or other modules if needed).
func AddSudo(jid string) {
	normalized := normalizeJIDForSudo(jid)
	sudoMu.Lock()
	defer sudoMu.Unlock()
	sudoUsers[normalized] = true
}

// RemoveSudo removes a JID from the sudo list.
func RemoveSudo(jid string) {
	normalized := normalizeJIDForSudo(jid)
	sudoMu.Lock()
	defer sudoMu.Unlock()
	delete(sudoUsers, normalized)
}

func normalizeJIDForSudo(jid string) string {
	// remove device ID part: "number:device@server" -> "number@server"
	if idx := strings.Index(jid, ":"); idx != -1 {
		at := strings.Index(jid, "@")
		if at > idx {
			return jid[:idx] + jid[at:]
		}
	}
	return jid
}

func addSudoHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .addsudo @mention or .addsudo <number>\nExample: .addsudo 923001234567")
	}

	normalized := normalizeJIDForSudo(target)
	sudoMu.Lock()
	sudoUsers[normalized] = true
	sudoMu.Unlock()

	phone := strings.Split(normalized, "@")[0]
	return ReplyErr(v, fmt.Sprintf("✅ Added *+%s* to sudo list.", phone))
}

func delSudoHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: .delsudo @mention or .delsudo <number>")
	}

	normalized := normalizeJIDForSudo(target)
	sudoMu.Lock()
	_, existed := sudoUsers[normalized]
	delete(sudoUsers, normalized)
	sudoMu.Unlock()

	phone := strings.Split(normalized, "@")[0]
	if !existed {
		return ReplyErr(v, fmt.Sprintf("⚠️ *+%s* was not in sudo list.", phone))
	}
	return ReplyErr(v, fmt.Sprintf("✅ Removed *+%s* from sudo list.", phone))
}

func sudoListHandler(v *events.Message, args []string) error {
	sudoMu.RLock()
	defer sudoMu.RUnlock()

	if len(sudoUsers) == 0 {
		return ReplyErr(v, "ℹ️ No sudo users configured.")
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("👑 *Sudo Users (%d):*", len(sudoUsers)))
	i := 1
	for jid := range sudoUsers {
		phone := strings.Split(jid, "@")[0]
		lines = append(lines, fmt.Sprintf("  %d. +%s", i, phone))
		i++
	}

	return ReplyErr(v, strings.Join(lines, "\n"))
}

func init() {
	Register(ModuleInfo{
		Name:        "Sudo",
		Description: "Manage sudo (elevated) users who can run protected commands",
		Commands: []CommandInfo{
			{Pattern: "addsudo", Handler: addSudoHandler, Sudo: true},
			{Pattern: "delsudo", Handler: delSudoHandler, Sudo: true},
			{Pattern: "sudolist", Handler: sudoListHandler, Sudo: true},
		},
	})
}
