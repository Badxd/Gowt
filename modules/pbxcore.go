package modules

import (
	"context"
	"strings"
	"time"

	"PbxGb/client"
	"PbxGb/config"
	"PbxGb/database"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// ── Branding ────────────────────────────────────────────────────────────────

const (
	BotName      = "PBX Elite"
	BotNameFancy = "Pʙx Eʟɪᴛᴇ"
	BotVersion   = "7.0.0"

	ChatURL   = "https://t.me/PBXCHATS"
	UpdateURL = "https://t.me/PBX_UPDATE"
	OwnerURL  = "https://t.me/BADMUNDAXD"
)

// StartTime is used by .ping / .alive and by the web dashboard.
var StartTime = time.Now()

var scMap = map[rune]rune{
	'a': 'ᴀ', 'b': 'ʙ', 'c': 'ᴄ', 'd': 'ᴅ', 'e': 'ᴇ', 'f': 'ꜰ', 'g': 'ɢ', 'h': 'ʜ', 'i': 'ɪ',
	'j': 'ᴊ', 'k': 'ᴋ', 'l': 'ʟ', 'm': 'ᴍ', 'n': 'ɴ', 'o': 'ᴏ', 'p': 'ᴘ', 'q': 'ǫ', 'r': 'ʀ',
	's': 'ꜱ', 't': 'ᴛ', 'u': 'ᴜ', 'v': 'ᴠ', 'w': 'ᴡ', 'x': 'x', 'y': 'ʏ', 'z': 'ᴢ',
}

// smallCaps converts text to PBX Elite's small-caps style.
func smallCaps(s string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(s) {
		if m, ok := scMap[r]; ok {
			sb.WriteRune(m)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// FormatUptime renders a duration as "1d 2h 3m 4s".
func FormatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	h := int(d.Hours()) % 24
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	switch {
	case days > 0:
		return strings.Join([]string{itoa(days) + "d", itoa(h) + "h", itoa(m) + "m", itoa(s) + "s"}, " ")
	case h > 0:
		return strings.Join([]string{itoa(h) + "h", itoa(m) + "m", itoa(s) + "s"}, " ")
	case m > 0:
		return itoa(m) + "m " + itoa(s) + "s"
	}
	return itoa(s) + "s"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

// CommandCount returns how many commands are registered.
func CommandCount() int {
	n := 0
	for _, m := range allModules {
		n += len(m.Commands)
	}
	return n
}

// ── JID helpers (LID-safe) ──────────────────────────────────────────────────
//
// WhatsApp now delivers many DMs from "@lid" addresses instead of
// "number@s.whatsapp.net". The old code rebuilt a JID from Sender.User with
// the phone-number server, which produced a WRONG address for LID senders —
// that is why AFK / PM-permit replies and blocks silently did nothing.
// Rules used now:
//   * reply to v.Info.Chat (always the correct, addressable JID)
//   * store/look up records by a canonical phone-number key (senderKey)

func lidToPN(lid types.JID) string {
	wac := client.Get()
	if wac == nil || wac.Store == nil || wac.Store.LIDs == nil {
		return ""
	}
	pn, err := wac.Store.LIDs.GetPNForLID(context.Background(), lid.ToNonAD())
	if err != nil || pn.IsEmpty() {
		return ""
	}
	return pn.ToNonAD().String()
}

// senderKey returns the canonical "<number>@s.whatsapp.net" of the sender.
func senderKey(v *events.Message) string {
	s := v.Info.Sender.ToNonAD()
	if s.Server == types.DefaultUserServer {
		return s.String()
	}
	if !v.Info.SenderAlt.IsEmpty() && v.Info.SenderAlt.Server == types.DefaultUserServer {
		return v.Info.SenderAlt.ToNonAD().String()
	}
	if s.Server == types.HiddenUserServer {
		if pn := lidToPN(s); pn != "" {
			return pn
		}
	}
	return s.String()
}

// canonKey normalises any user-supplied / quoted JID string to the same form.
func canonKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "@") {
		raw = strings.TrimPrefix(raw, "+") + "@s.whatsapp.net"
	}
	jid, err := types.ParseJID(raw)
	if err != nil {
		return raw
	}
	jid = jid.ToNonAD()
	if jid.Server == types.HiddenUserServer {
		if pn := lidToPN(jid); pn != "" {
			return pn
		}
	}
	return jid.String()
}

func chatKey(chat types.JID) string { return canonKey(chat.String()) }

// isSudoSender checks sudo by every form the sender can appear in.
func isSudoSender(v *events.Message) bool {
	return database.IsSudo(v.Info.Sender.String()) || database.IsSudo(senderKey(v))
}

// isStale — backlog delivered after a reconnect must not trigger AFK / PM warnings.
func isStale(v *events.Message) bool {
	return !v.Info.Timestamp.IsZero() && time.Since(v.Info.Timestamp) > 2*time.Minute
}

// isRealMessage filters reactions / protocol (edit, revoke, history) messages.
func isRealMessage(v *events.Message) bool {
	m := v.Message
	if m == nil {
		return false
	}
	if m.GetProtocolMessage() != nil || m.GetReactionMessage() != nil {
		return false
	}
	return true
}

// autoApproveOnOutgoing — classic PM-permit behaviour: when the OWNER writes to
// someone first (not a command), that chat is approved automatically.
func autoApproveOnOutgoing(v *events.Message) {
	pmMu.RLock()
	enabled := pmEnabled
	pmMu.RUnlock()
	if !enabled {
		return
	}
	if strings.HasPrefix(extractText(v), config.C.Prefix) {
		return
	}
	wac := client.Get()
	if wac.Store.ID != nil && v.Info.Chat.User == wac.Store.ID.User {
		return // own "saved messages" chat
	}
	if wac.Store.LID.User != "" && v.Info.Chat.User == wac.Store.LID.User {
		return
	}
	key := chatKey(v.Info.Chat)
	rec := database.GetPMRecord(key)
	if rec.Approved || rec.Blocked {
		return
	}
	_ = database.UpsertPMRecord(key, true, false, 0)
}
