package modules

// ── CUSTOM RAID MODULE ───────────────────────────────────────────────────────
// Ports: customraid.py
//
// Each bot owner maintains their own list of raid strings (max 50, 200 chars each).
// Stored in memory per session.
// Commands:
//   .addraid <text>   / .addr           — add string to personal list
//   .rmraid <n>       / .rmr <n>        — remove by 1-based index
//   .listraid / .raidlist               — show personal list
//   .delraid  / .clearraid              — clear entire list
//   .myraid <count> <phone>             — raid with personal list
//   .customraid / .craid                — aliases for .myraid

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow/types/events"
)

func init() {
	Register(ModuleInfo{
		Name:        "CustomRaid",
		Description: "Personal custom raid message list — add, manage, and raid with your own strings",
		Commands: []CommandInfo{
			{Pattern: "addraid",    Handler: addRaidHandler,    Sudo: true},
			{Pattern: "addr",       Handler: addRaidHandler,    Sudo: true},
			{Pattern: "rmraid",     Handler: rmRaidHandler,     Sudo: true},
			{Pattern: "rmr",        Handler: rmRaidHandler,     Sudo: true},
			{Pattern: "listraid",   Handler: listRaidHandler,   Sudo: true},
			{Pattern: "raidlist",   Handler: listRaidHandler,   Sudo: true},
			{Pattern: "delraid",    Handler: delRaidHandler,    Sudo: true},
			{Pattern: "clearraid",  Handler: delRaidHandler,    Sudo: true},
			{Pattern: "myraid",     Handler: myRaidHandler,     Sudo: true},
			{Pattern: "customraid", Handler: myRaidHandler,     Sudo: true},
			{Pattern: "craid",      Handler: myRaidHandler,     Sudo: true},
		},
	})
}

const (
	maxCustomRaidItems   = 50
	maxCustomRaidItemLen = 200
)

var (
	customRaidMu    sync.RWMutex
	customRaidLists = make(map[string][]string)
)

func customRaidOwnerKey(v *events.Message) string {
	return v.Info.Sender.ToNonAD().String()
}

func addRaidHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		safeSendMessage(v, "Usage: .addraid <text>")
		return nil
	}
	text := strings.Join(args, " ")
	if len(text) > maxCustomRaidItemLen {
		safeSendMessage(v, fmt.Sprintf("Text too long. Max %d characters.", maxCustomRaidItemLen))
		return nil
	}

	key := customRaidOwnerKey(v)
	customRaidMu.Lock()
	list := customRaidLists[key]
	if len(list) >= maxCustomRaidItems {
		customRaidMu.Unlock()
		safeSendMessage(v, fmt.Sprintf("List full (%d/%d). Remove with .rmraid first.", len(list), maxCustomRaidItems))
		return nil
	}
	customRaidLists[key] = append(list, text)
	total := len(customRaidLists[key])
	customRaidMu.Unlock()

	safeSendMessage(v, fmt.Sprintf("✅ Added! Total: %d/%d", total, maxCustomRaidItems))
	return nil
}

func rmRaidHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		safeSendMessage(v, "Usage: .rmraid <number>")
		return nil
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n < 1 {
		safeSendMessage(v, "Provide a valid item number.")
		return nil
	}

	key := customRaidOwnerKey(v)
	customRaidMu.Lock()
	list := customRaidLists[key]
	if n > len(list) {
		customRaidMu.Unlock()
		safeSendMessage(v, fmt.Sprintf("No item #%d. List has %d items.", n, len(list)))
		return nil
	}
	removed := list[n-1]
	customRaidLists[key] = append(list[:n-1], list[n:]...)
	customRaidMu.Unlock()

	safeSendMessage(v, fmt.Sprintf("✅ Removed #%d: %s", n, removed))
	return nil
}

func listRaidHandler(v *events.Message, args []string) error {
	key := customRaidOwnerKey(v)
	customRaidMu.RLock()
	list := append([]string{}, customRaidLists[key]...)
	customRaidMu.RUnlock()

	if len(list) == 0 {
		safeSendMessage(v, "Your custom raid list is empty.\nUse .addraid <text> to add strings.")
		return nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*Your Custom Raid List* (%d/%d):\n", len(list), maxCustomRaidItems))
	for i, item := range list {
		display := item
		if len(display) > 60 {
			display = display[:57] + "..."
		}
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, display))
	}
	safeSendMessage(v, sb.String())
	return nil
}

func delRaidHandler(v *events.Message, args []string) error {
	key := customRaidOwnerKey(v)
	customRaidMu.Lock()
	delete(customRaidLists, key)
	customRaidMu.Unlock()

	safeSendMessage(v, "✅ Custom raid list cleared.")
	return nil
}

func myRaidHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		safeSendMessage(v, "Usage: .myraid <count> <phone>")
		return nil
	}

	key := customRaidOwnerKey(v)
	customRaidMu.RLock()
	list := append([]string{}, customRaidLists[key]...)
	customRaidMu.RUnlock()

	if len(list) == 0 {
		safeSendMessage(v, "Your custom raid list is empty. Add strings with .addraid first.")
		return nil
	}

	count, err := strconv.Atoi(args[0])
	if err != nil || count <= 0 {
		safeSendMessage(v, "Count must be a positive number.")
		return nil
	}
	if count > 500 {
		count = 500
	}

	jid, name := resolveMentionOrPhone(v, args[1])
	if jid.IsEmpty() {
		safeSendMessage(v, "Could not resolve user: "+args[1])
		return nil
	}

	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, list, count, jid, name)
	return nil
}
