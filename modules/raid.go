package modules

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// ── RAID MODULE ─────────────────────────────────────────────────────────────
// Ports: raid.py, multiraid functionality
// All pools live in raidpools.go

func init() {
	Register(ModuleInfo{
		Name:        "Raid",
		Description: "Flood a user with messages from various pools (raid, pbiraid, hiraid, flirt, bday, shayri, emoji, oneword …)",
		Commands: []CommandInfo{
			{Pattern: "raid",      Handler: raidHandler,       Sudo: true},
			{Pattern: "pbiraid",   Handler: pbiraidHandler,    Sudo: true},
			{Pattern: "hiraid",    Handler: hiraidHandler,     Sudo: true},
			{Pattern: "imoji",     Handler: emojiRaidHandler,  Sudo: true},
			{Pattern: "oneword",   Handler: onewordHandler,    Sudo: true},
			{Pattern: "shraid",    Handler: shraidHandler,     Sudo: true},
			{Pattern: "pbishraid", Handler: pbishraidHandler,  Sudo: true},
			{Pattern: "engshraid", Handler: engshraidHandler,  Sudo: true},
			{Pattern: "hishraid",  Handler: hishraidHandler,   Sudo: true},
			{Pattern: "flirt",     Handler: flirtHandler,      Sudo: true},
			{Pattern: "pflirt",    Handler: pflirtHandler,     Sudo: true},
			{Pattern: "hflirt",    Handler: hflirtHandler,     Sudo: true},
			{Pattern: "eflirt",    Handler: eflirtHandler,     Sudo: true},
			{Pattern: "bday",      Handler: bdayHandler,       Sudo: true},
			{Pattern: "pbday",     Handler: pbdayHandler,      Sudo: true},
			{Pattern: "hbday",     Handler: hbdayHandler,      Sudo: true},
			{Pattern: "ebday",     Handler: ebdayHandler,      Sudo: true},
			{Pattern: "multiraid", Handler: multiraidHandler,  Sudo: true},
		},
	})
}

// ── HELPERS ─────────────────────────────────────────────────────────────────

func randItem(pool []string) string {
	if len(pool) == 0 {
		return ""
	}
	return pool[rand.Intn(len(pool))]
}

// parseRaidArgs parses "<count> <@mention|phone>" from args.
func parseRaidArgs(v *events.Message, args []string) (int, types.JID, string, error) {
	if len(args) < 2 {
		return 0, types.JID{}, "", fmt.Errorf("Usage: .cmd <count> <phone>")
	}
	count, err := strconv.Atoi(args[0])
	if err != nil || count <= 0 {
		return 0, types.JID{}, "", fmt.Errorf("Count must be a positive number")
	}
	if count > 500 {
		count = 500
	}

	jid, name := resolveMentionOrPhone(v, args[1])
	if jid.IsEmpty() {
		return 0, types.JID{}, "", fmt.Errorf("Could not resolve user: %s", args[1])
	}
	return count, jid, name, nil
}

// resolveMentionOrPhone resolves phone number string to JID + display name.
func resolveMentionOrPhone(v *events.Message, raw string) (types.JID, string) {
	wac := getClientSafe()
	if wac == nil {
		return types.JID{}, ""
	}

	// Strip @ prefix
	phone := strings.TrimPrefix(raw, "@")
	phone = strings.TrimPrefix(phone, "+")
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")

	// Must be numeric
	for _, c := range phone {
		if c < '0' || c > '9' {
			return types.JID{}, ""
		}
	}
	if len(phone) < 7 {
		return types.JID{}, ""
	}

	jid := types.NewJID(phone, types.DefaultUserServer)
	return jid, phone
}

// doRaid sends `count` messages from pool, mentioning targetJID.
// Rate-limited: 50ms between sends, 2s backoff on error.
func doRaid(wac *whatsmeow.Client, v *events.Message, pool []string, count int, targetJID types.JID, targetName string) {
	chatJID := v.Info.Chat

	// Best-effort delete the trigger message
	_ = wac.RevokeMessage(chatJID, v.Info.ID)

	for i := 0; i < count; i++ {
		msg := "@" + targetJID.User + " " + randItem(pool)

		protoMsg := &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: strPtr(msg),
				ContextInfo: &waProto.ContextInfo{
					MentionedJID: []string{targetJID.String()},
				},
			},
		}

		ctx, cancel := newContext(10)
		_, err := wac.SendMessage(ctx, chatJID, protoMsg)
		cancel()

		if err != nil {
			time.Sleep(2 * time.Second)
		} else {
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// ── RAID COMMAND HANDLERS ───────────────────────────────────────────────────

func raidHandler(v *events.Message, args []string) error {
	count, jid, name, err := parseRaidArgs(v, args)
	if err != nil {
		safeSendMessage(v, err.Error())
		return nil
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, RAID_POOL, count, jid, name)
	return nil
}

func pbiraidHandler(v *events.Message, args []string) error {
	count, jid, name, err := parseRaidArgs(v, args)
	if err != nil {
		safeSendMessage(v, err.Error())
		return nil
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, PBIRAID_POOL, count, jid, name)
	return nil
}

func hiraidHandler(v *events.Message, args []string) error {
	count, jid, name, err := parseRaidArgs(v, args)
	if err != nil {
		safeSendMessage(v, err.Error())
		return nil
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, HIRAID_POOL, count, jid, name)
	return nil
}

func emojiRaidHandler(v *events.Message, args []string) error {
	count, jid, name, err := parseRaidArgs(v, args)
	if err != nil {
		safeSendMessage(v, err.Error())
		return nil
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, EMOJI_POOL, count, jid, name)
	return nil
}

func onewordHandler(v *events.Message, args []string) error {
	count, jid, name, err := parseRaidArgs(v, args)
	if err != nil {
		safeSendMessage(v, err.Error())
		return nil
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, ONEWORD_POOL, count, jid, name)
	return nil
}

func shraidHandler(v *events.Message, args []string) error {
	count, jid, name, err := parseRaidArgs(v, args)
	if err != nil {
		safeSendMessage(v, err.Error())
		return nil
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, SHAYRI_POOL, count, jid, name)
	return nil
}

func pbishraidHandler(v *events.Message, args []string) error {
	count, jid, name, err := parseRaidArgs(v, args)
	if err != nil {
		safeSendMessage(v, err.Error())
		return nil
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, PBI_SHAYRI_POOL, count, jid, name)
	return nil
}

func engshraidHandler(v *events.Message, args []string) error {
	count, jid, name, err := parseRaidArgs(v, args)
	if err != nil {
		safeSendMessage(v, err.Error())
		return nil
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, ENG_SHAYRI_POOL, count, jid, name)
	return nil
}

func hishraidHandler(v *events.Message, args []string) error {
	count, jid, name, err := parseRaidArgs(v, args)
	if err != nil {
		safeSendMessage(v, err.Error())
		return nil
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, HI_SHAYRI_POOL, count, jid, name)
	return nil
}

func flirtHandler(v *events.Message, args []string) error {
	count, jid, name, err := parseRaidArgs(v, args)
	if err != nil {
		safeSendMessage(v, err.Error())
		return nil
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, FLIRT_POOL, count, jid, name)
	return nil
}

func pflirtHandler(v *events.Message, args []string) error {
	count, jid, name, err := parseRaidArgs(v, args)
	if err != nil {
		safeSendMessage(v, err.Error())
		return nil
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, PFLIRT_POOL, count, jid, name)
	return nil
}

func hflirtHandler(v *events.Message, args []string) error {
	count, jid, name, err := parseRaidArgs(v, args)
	if err != nil {
		safeSendMessage(v, err.Error())
		return nil
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, HFLIRT_POOL, count, jid, name)
	return nil
}

func eflirtHandler(v *events.Message, args []string) error {
	count, jid, name, err := parseRaidArgs(v, args)
	if err != nil {
		safeSendMessage(v, err.Error())
		return nil
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, EFLIRT_POOL, count, jid, name)
	return nil
}

func bdayHandler(v *events.Message, args []string) error {
	count, jid, name, err := parseRaidArgs(v, args)
	if err != nil {
		safeSendMessage(v, err.Error())
		return nil
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, BDAY_POOL, count, jid, name)
	return nil
}

func pbdayHandler(v *events.Message, args []string) error {
	count, jid, name, err := parseRaidArgs(v, args)
	if err != nil {
		safeSendMessage(v, err.Error())
		return nil
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, PBDAY_POOL, count, jid, name)
	return nil
}

func hbdayHandler(v *events.Message, args []string) error {
	count, jid, name, err := parseRaidArgs(v, args)
	if err != nil {
		safeSendMessage(v, err.Error())
		return nil
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, HBDAY_POOL, count, jid, name)
	return nil
}

func ebdayHandler(v *events.Message, args []string) error {
	count, jid, name, err := parseRaidArgs(v, args)
	if err != nil {
		safeSendMessage(v, err.Error())
		return nil
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}
	go doRaid(wac, v, EBDAY_POOL, count, jid, name)
	return nil
}

// multiraidHandler — .multiraid <count> <phone1> <phone2> ...
func multiraidHandler(v *events.Message, args []string) error {
	if len(args) < 3 {
		safeSendMessage(v, "Usage: .multiraid <count> <phone1> <phone2> ...")
		return nil
	}
	count, err := strconv.Atoi(args[0])
	if err != nil || count <= 0 {
		safeSendMessage(v, "Count must be a positive number")
		return nil
	}
	if count > 200 {
		count = 200
	}
	wac := getClientSafe()
	if wac == nil {
		return nil
	}

	for _, raw := range args[1:] {
		jid, name := resolveMentionOrPhone(v, raw)
		if jid.IsEmpty() {
			continue
		}
		go doRaid(wac, v, RAID_POOL, count, jid, name)
	}
	return nil
}
