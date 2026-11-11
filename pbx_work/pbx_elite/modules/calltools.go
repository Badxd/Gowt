package modules

// Call Tools — WhatsApp call utilities using whatsmeow.
//
// Commands:
//   .callinfo        — info about the last incoming call (caller, time, type)
//   .calljoin        — attempt to join an active group call (sends join signal)
//   .callleave       — leave current active call
//   .callhistory     — show last 10 calls received this session
//   .autocallreject  — already in callreject.go — this adds per-contact allow
//   .vcping          — ping all group members who are in an active call
//
// Note: WhatsApp does NOT expose a "join existing call" API in whatsmeow.
// What we CAN do:
//   - Track incoming CallOffer events
//   - Show who called, when, audio/video
//   - RejectCall
//   - Notify group when a call starts (auto-ping feature)

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"PbxGb/client"
)

// ── CALL HISTORY STORE ────────────────────────────────────────────────────────
type callRecord struct {
	Caller    types.JID
	Chat      types.JID
	CallID    string
	IsVideo   bool
	IsGroup   bool
	Time      time.Time
	Ended     bool
}

var (
	callHistoryMu sync.RWMutex
	callHistory   []callRecord // last 50 calls
)

// TrackCallOffer — called from main event loop on events.CallOffer
func TrackCallOffer(e *events.CallOffer) {
	callHistoryMu.Lock()
	defer callHistoryMu.Unlock()
	rec := callRecord{
		Caller:  e.CallCreator,
		Chat:    e.CallCreator, // for DM calls
		CallID:  e.CallID,
		IsVideo: e.IsVideo,
		IsGroup: false,
		Time:    time.Now(),
	}
	callHistory = append(callHistory, rec)
	if len(callHistory) > 50 {
		callHistory = callHistory[len(callHistory)-50:]
	}
	// also update vccount tracker
	RegisterCallEvent(e.CallCreator.String(), e.CallCreator.String(), true)
}

// TrackCallTerminate — called from main event loop on events.CallTerminate
func TrackCallTerminate(e *events.CallTerminate) {
	callHistoryMu.Lock()
	defer callHistoryMu.Unlock()
	for i := range callHistory {
		if callHistory[i].CallID == e.CallID {
			callHistory[i].Ended = true
		}
	}
	RegisterCallEvent(e.CallCreator.String(), e.CallCreator.String(), false)
}

// ── CALL HISTORY COMMAND ──────────────────────────────────────────────────────
func callHistoryHandler(v *events.Message, args []string) error {
	callHistoryMu.RLock()
	defer callHistoryMu.RUnlock()

	if len(callHistory) == 0 {
		return ReplyErr(v, "📵 No calls recorded this session.")
	}

	// show last 10, newest first
	start := len(callHistory) - 10
	if start < 0 {
		start = 0
	}
	slice := callHistory[start:]

	lines := []string{fmt.Sprintf("📞 *Last %d call(s) this session:*\n", len(slice))}
	for i := len(slice) - 1; i >= 0; i-- {
		r := slice[i]
		kind := "🎙️ Audio"
		if r.IsVideo {
			kind = "📹 Video"
		}
		status := "📲 Incoming"
		if r.Ended {
			status = "📵 Ended"
		}
		lines = append(lines, fmt.Sprintf(
			"%s call from *+%s*\n   %s | %s",
			kind, r.Caller.User, status, r.Time.Format("02 Jan 3:04 PM"),
		))
	}
	return ReplyErr(v, strings.Join(lines, "\n\n"))
}

// ── LAST CALL INFO ────────────────────────────────────────────────────────────
func callInfoHandler(v *events.Message, args []string) error {
	callHistoryMu.RLock()
	defer callHistoryMu.RUnlock()

	if len(callHistory) == 0 {
		return ReplyErr(v, "📵 No calls recorded this session.")
	}
	r := callHistory[len(callHistory)-1]
	kind := "🎙️ Audio"
	if r.IsVideo {
		kind = "📹 Video"
	}
	status := "📲 Active / Missed"
	if r.Ended {
		status = "📵 Ended"
	}
	return ReplyErr(v, fmt.Sprintf(
		"📞 *Last Call Info*\n\n"+
			"👤 *Caller:* +%s\n"+
			"📱 *Type:* %s\n"+
			"⏰ *Time:* %s\n"+
			"📊 *Status:* %s\n"+
			"🆔 *Call ID:* `%s`",
		r.Caller.User,
		kind,
		r.Time.Format("02 Jan 2006, 3:04 PM"),
		status,
		r.CallID,
	))
}

// ── CALL BACK ─────────────────────────────────────────────────────────────────
// Sends a text message to the last caller saying "calling back"
func callBackHandler(v *events.Message, args []string) error {
	callHistoryMu.RLock()
	if len(callHistory) == 0 {
		callHistoryMu.RUnlock()
		return ReplyErr(v, "📵 No recent call to call back.")
	}
	r := callHistory[len(callHistory)-1]
	callHistoryMu.RUnlock()

	wac := client.Get()
	msg := "📞 Calling you back..."
	if len(args) > 0 {
		msg = strings.Join(args, " ")
	}
	sendText(wac, r.Caller, msg)
	return ReplyErr(v, fmt.Sprintf("✅ Sent callback message to *+%s*.", r.Caller.User))
}

// ── REJECT LAST CALL ─────────────────────────────────────────────────────────
func rejectLastCallHandler(v *events.Message, args []string) error {
	callHistoryMu.RLock()
	if len(callHistory) == 0 {
		callHistoryMu.RUnlock()
		return ReplyErr(v, "📵 No recent call to reject.")
	}
	r := callHistory[len(callHistory)-1]
	callHistoryMu.RUnlock()

	if r.Ended {
		return ReplyErr(v, "ℹ️ Call already ended.")
	}

	wac := client.Get()
	if err := wac.RejectCall(v.Info.Chat.String(), r.Caller, r.CallID); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Reject failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("🚫 Rejected call from *+%s*.", r.Caller.User))
}

// ── VC PING (notify group members about active call) ─────────────────────────
func vcPingHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ Only works in groups.")
	}
	wac := client.Get()
	info, err := wac.GetGroupInfo(v.Info.Chat)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	msg := "📞 *Voice/Video Call is LIVE in this group!*\nJoin now 🎙️"
	if len(args) > 0 {
		msg = strings.Join(args, " ")
	}

	// mention all members
	var mentions []string
	for _, p := range info.Participants {
		mentions = append(mentions, "@"+p.JID.User)
	}
	full := msg + "\n\n" + strings.Join(mentions, " ")
	sendTextWithMentions(wac, v.Info.Chat, full, info.Participants)
	return ReplyErr(v, fmt.Sprintf("📢 Pinged *%d* members about the call.", len(info.Participants)))
}

func init() {
	Register(ModuleInfo{
		Name:        "CallTools",
		Description: "Call history, call info, reject last call, callback message, vc ping",
		Commands: []CommandInfo{
			{Pattern: "callhistory", Handler: callHistoryHandler,    Sudo: true},
			{Pattern: "calllog",     Handler: callHistoryHandler,    Sudo: true},
			{Pattern: "callinfo",    Handler: callInfoHandler,       Sudo: true},
			{Pattern: "lastcall",    Handler: callInfoHandler,       Sudo: true},
			{Pattern: "callback",    Handler: callBackHandler,       Sudo: true},
			{Pattern: "rejectlast",  Handler: rejectLastCallHandler, Sudo: true},
			{Pattern: "vcping",      Handler: vcPingHandler,         Sudo: true},
		},
	})
}
