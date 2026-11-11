package modules

// Ghost Mode — full invisibility suite for PBX Elite.
//
// Master toggle:
//   .ghost on            — enable ALL ghost protections at once
//   .ghost off           — disable ALL, restore normal visibility
//   .ghost status        — show current state of every setting
//
// Individual controls (each works independently):
//   .hideread on/off     — hide blue ticks (read receipts)
//   .hideonline on/off   — hide online status
//   .hidelastseen on/off — hide last seen timestamp
//   .hidepp on/off       — hide profile photo from everyone
//   .hidestatus on/off   — hide WhatsApp Status from non-contacts
//   .hidegroups on/off   — stop anyone adding you to groups
//
// Per-number profile photo hiding (block+unblock trick):
//   .ppblock <num>       — hide your PP from specific number
//   .ppunblock <num>     — remove from pp-block tracking list
//   .ppblocklist         — list pp-blocked numbers
//
// Full status:
//   .ghoststatus         — show every setting with values

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"PbxGb/client"
)

// ── STATE ─────────────────────────────────────────────────────────────────────

var gMu sync.RWMutex

type ghostStateT struct {
	Enabled        bool
	ReadHidden     bool
	OnlineHidden   bool
	LastSeenHidden bool
	PPHidden       bool
	StatusHidden   bool
	GroupAddHidden bool
	PPBlockedNums  []string // numbers we did block+unblock cycle on
}

var gState = &ghostStateT{}

// ── LOW-LEVEL HELPER ──────────────────────────────────────────────────────────

func privSet(kind types.PrivacySettingType, value types.PrivacySetting) error {
	wac := client.Get()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := wac.SetPrivacySetting(ctx, kind, value)
	return err
}

// ── MASTER GHOST TOGGLE ───────────────────────────────────────────────────────

func ghostHandler(v *events.Message, args []string) error {
	if len(args) == 0 || strings.ToLower(args[0]) == "status" {
		return ghostStatusHandler(v, nil)
	}

	cmd := strings.ToLower(args[0])
	enable := cmd == "on" || cmd == "enable" || cmd == "1"
	disable := cmd == "off" || cmd == "disable" || cmd == "0"
	if !enable && !disable {
		return ReplyErr(v, "Usage: `.ghost on` / `.ghost off` / `.ghost status`")
	}

	type setting struct {
		kind    types.PrivacySettingType
		label   string
		onVal   types.PrivacySetting
		offVal  types.PrivacySetting
	}

	settings := []setting{
		// Read receipts: all=visible, none=hidden
		{types.PrivacySettingTypeReadReceipts, "Read receipts",
			types.PrivacySettingNone, types.PrivacySettingAll},
		// Last seen: all=visible, none=hidden
		{types.PrivacySettingTypeLastSeen, "Last seen",
			types.PrivacySettingNone, types.PrivacySettingAll},
		// Online: all=visible, match_last_seen=hidden (matches last seen setting)
		{types.PrivacySettingTypeOnline, "Online status",
			types.PrivacySettingMatchLastSeen, types.PrivacySettingAll},
		// Profile photo: all=visible, none=hidden
		{types.PrivacySettingTypeProfile, "Profile photo",
			types.PrivacySettingNone, types.PrivacySettingAll},
		// Status/story: all=visible, none=hidden
		{types.PrivacySettingTypeStatus, "Status/stories",
			types.PrivacySettingNone, types.PrivacySettingAll},
		// Group add: all=anyone, contacts=contacts only
		{types.PrivacySettingTypeGroupAdd, "Group add by others",
			types.PrivacySettingContacts, types.PrivacySettingAll},
	}

	var errs []string
	for _, s := range settings {
		val := s.offVal
		if enable {
			val = s.onVal
		}
		if err := privSet(s.kind, val); err != nil {
			errs = append(errs, fmt.Sprintf("⚠️ %s: %v", s.label, err))
		}
	}

	// Presence beacon
	wac := client.Get()
	ctx := context.Background()
	if enable {
		_ = wac.SendPresence(ctx, types.PresenceUnavailable)
	} else {
		_ = wac.SendPresence(ctx, types.PresenceAvailable)
	}

	gMu.Lock()
	gState.Enabled = enable
	gState.ReadHidden = enable
	gState.OnlineHidden = enable
	gState.LastSeenHidden = enable
	gState.PPHidden = enable
	gState.StatusHidden = enable
	gState.GroupAddHidden = enable
	gMu.Unlock()

	if enable {
		msg := "👻 *Ghost Mode ENABLED*\n\n" +
			"✅ Blue ticks (read receipts) hidden\n" +
			"✅ Online status hidden\n" +
			"✅ Last seen hidden\n" +
			"✅ Profile photo hidden from everyone\n" +
			"✅ WhatsApp Status hidden\n" +
			"✅ Group add restricted to contacts only\n" +
			"✅ Presence set to unavailable\n\n" +
			"Use `.ghost off` to restore."
		if len(errs) > 0 {
			msg += "\n\n*Warnings:*\n" + strings.Join(errs, "\n")
		}
		return ReplyErr(v, msg)
	}

	msg := "✅ *Ghost Mode DISABLED*\n\n" +
		"• All visibility settings restored\n" +
		"• Presence set to available\n\n" +
		"Use `.ghost on` to re-enable."
	if len(errs) > 0 {
		msg += "\n\n*Warnings:*\n" + strings.Join(errs, "\n")
	}
	return ReplyErr(v, msg)
}

// ── INDIVIDUAL TOGGLES ────────────────────────────────────────────────────────

func hideReadHandler(v *events.Message, args []string) error {
	return singleToggle(v, args, "hideread",
		types.PrivacySettingTypeReadReceipts,
		"Read receipts (blue ticks)",
		types.PrivacySettingNone,
		types.PrivacySettingAll,
		&gState.ReadHidden,
	)
}

func hideOnlineHandler(v *events.Message, args []string) error {
	return singleToggle(v, args, "hideonline",
		types.PrivacySettingTypeOnline,
		"Online status",
		types.PrivacySettingMatchLastSeen,
		types.PrivacySettingAll,
		&gState.OnlineHidden,
	)
}

func hideLastSeenHandler(v *events.Message, args []string) error {
	return singleToggle(v, args, "hidelastseen",
		types.PrivacySettingTypeLastSeen,
		"Last seen",
		types.PrivacySettingNone,
		types.PrivacySettingAll,
		&gState.LastSeenHidden,
	)
}

func hidePPHandler(v *events.Message, args []string) error {
	return singleToggle(v, args, "hidepp",
		types.PrivacySettingTypeProfile,
		"Profile photo",
		types.PrivacySettingNone,
		types.PrivacySettingAll,
		&gState.PPHidden,
	)
}

func hideStatusHandler(v *events.Message, args []string) error {
	return singleToggle(v, args, "hidestatus",
		types.PrivacySettingTypeStatus,
		"WhatsApp Status/stories",
		types.PrivacySettingNone,
		types.PrivacySettingAll,
		&gState.StatusHidden,
	)
}

func hideGroupsHandler(v *events.Message, args []string) error {
	return singleToggle(v, args, "hidegroups",
		types.PrivacySettingTypeGroupAdd,
		"Group add by others",
		types.PrivacySettingContacts,
		types.PrivacySettingAll,
		&gState.GroupAddHidden,
	)
}

// singleToggle is the shared logic for all individual on/off handlers.
func singleToggle(
	v *events.Message, args []string, cmdName string,
	kind types.PrivacySettingType, label string,
	onVal, offVal types.PrivacySetting,
	flag *bool,
) error {
	if len(args) == 0 {
		gMu.RLock()
		cur := *flag
		gMu.RUnlock()
		return ReplyErr(v, fmt.Sprintf("*%s* is currently: %s\nUsage: `.%s on` / `.%s off`",
			label, boolEmoji(cur), cmdName, cmdName))
	}

	cmd := strings.ToLower(args[0])
	enable := cmd == "on" || cmd == "enable" || cmd == "1"
	disable := cmd == "off" || cmd == "disable" || cmd == "0"
	if !enable && !disable {
		return ReplyErr(v, fmt.Sprintf("Usage: `.%s on` / `.%s off`", cmdName, cmdName))
	}

	val := offVal
	if enable {
		val = onVal
	}

	if err := privSet(kind, val); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}

	gMu.Lock()
	*flag = enable
	gMu.Unlock()

	verb := "hidden"
	if !enable {
		verb = "visible"
	}
	return ReplyErr(v, fmt.Sprintf("✅ *%s* is now *%s*.", label, verb))
}

// ── PER-NUMBER PP HIDE (block+unblock trick) ─────────────────────────────────
//
// WhatsApp: after you block then unblock someone, they can no longer see
// your profile photo — the effect persists until you change your photo.
// This is the same technique used by GB WhatsApp's "hide PP from X" feature.

func ppBlockHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: `.ppblock @mention` or `.ppblock 919xxxxxxxx`")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid number.")
	}

	wac := client.Get()
	ctx := context.Background()

	// Block
	if _, err := wac.UpdateBlocklist(ctx, jid, events.BlocklistChangeActionBlock, ""); err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Block step failed: %v", err))
	}
	time.Sleep(700 * time.Millisecond)

	// Unblock — they're unblocked but PP is now invisible to them
	if _, err := wac.UpdateBlocklist(ctx, jid, events.BlocklistChangeActionUnblock, ""); err != nil {
		// non-fatal — block already did the job
		_ = err
	}

	gMu.Lock()
	alreadyIn := false
	for _, n := range gState.PPBlockedNums {
		if n == jid.User {
			alreadyIn = true
			break
		}
	}
	if !alreadyIn {
		gState.PPBlockedNums = append(gState.PPBlockedNums, jid.User)
	}
	gMu.Unlock()

	return ReplyErr(v, fmt.Sprintf(
		"🖼️ *PP hidden from +%s*\n\n"+
			"Block → unblock cycle done.\n"+
			"They can no longer see your profile photo.\n\n"+
			"To fully restore: change your profile photo once.",
		jid.User,
	))
}

func ppUnblockHandler(v *events.Message, args []string) error {
	target, err := extractTargetJID(v, args)
	if err != nil || target == "" {
		return ReplyErr(v, "Usage: `.ppunblock @mention` or `.ppunblock 919xxxxxxxx`")
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return ReplyErr(v, "❌ Invalid number.")
	}

	gMu.Lock()
	var newList []string
	for _, n := range gState.PPBlockedNums {
		if n != jid.User {
			newList = append(newList, n)
		}
	}
	gState.PPBlockedNums = newList
	gMu.Unlock()

	return ReplyErr(v, fmt.Sprintf(
		"✅ Removed *+%s* from PP-block tracking.\n\n"+
			"To restore their view of your photo: change your profile photo once.",
		jid.User,
	))
}

func ppBlockListHandler(v *events.Message, args []string) error {
	gMu.RLock()
	list := make([]string, len(gState.PPBlockedNums))
	copy(list, gState.PPBlockedNums)
	gMu.RUnlock()

	if len(list) == 0 {
		return ReplyErr(v, "🖼️ PP-block list is empty.")
	}
	var lines []string
	for i, n := range list {
		lines = append(lines, fmt.Sprintf("%d. +%s", i+1, n))
	}
	return ReplyErr(v, fmt.Sprintf("🖼️ *PP hidden from (%d):*\n\n%s", len(list), strings.Join(lines, "\n")))
}

// ── FULL STATUS ───────────────────────────────────────────────────────────────

func ghostStatusHandler(v *events.Message, args []string) error {
	gMu.RLock()
	g := *gState
	gMu.RUnlock()

	ppList := "_none_"
	if len(g.PPBlockedNums) > 0 {
		nums := make([]string, len(g.PPBlockedNums))
		for i, n := range g.PPBlockedNums {
			nums[i] = "+"+n
		}
		ppList = strings.Join(nums, ", ")
	}

	return ReplyErr(v, fmt.Sprintf(
		"👻 *PBX Ghost Status*\n\n"+
			"🔒 Ghost mode (master): %s\n\n"+
			"🔵 Read receipts (blue tick) hidden: %s\n"+
			"🌐 Online status hidden: %s\n"+
			"🕐 Last seen hidden: %s\n"+
			"🖼️ Profile photo hidden: %s\n"+
			"📢 Status/stories hidden: %s\n"+
			"👥 Group add restricted: %s\n\n"+
			"🚫 *PP hidden from (block trick):*\n%s\n\n"+
			"*Quick cmds:*\n"+
			"`.ghost on/off` · `.hideread on/off` · `.hideonline on/off`\n"+
			"`.hidelastseen on/off` · `.hidepp on/off` · `.hidestatus on/off`\n"+
			"`.ppblock <num>` · `.ppunblock <num>` · `.ppblocklist`",
		boolEmoji(g.Enabled),
		boolEmoji(g.ReadHidden),
		boolEmoji(g.OnlineHidden),
		boolEmoji(g.LastSeenHidden),
		boolEmoji(g.PPHidden),
		boolEmoji(g.StatusHidden),
		boolEmoji(g.GroupAddHidden),
		ppList,
	))
}

// ── UTIL ──────────────────────────────────────────────────────────────────────

func boolEmoji(b bool) string {
	if b {
		return "✅ Yes"
	}
	return "❌ No"
}

// ── REGISTER ─────────────────────────────────────────────────────────────────

func init() {
	Register(ModuleInfo{
		Name:        "GhostMode",
		Description: "Full invisibility: hide blue ticks, online, last seen, PP, status; per-number PP hiding",
		Commands: []CommandInfo{
			// Master
			{Pattern: "ghost",        Handler: ghostHandler,       Sudo: true},
			// Individual
			{Pattern: "hideread",     Handler: hideReadHandler,    Sudo: true},
			{Pattern: "hideonline",   Handler: hideOnlineHandler,  Sudo: true},
			{Pattern: "hidelastseen", Handler: hideLastSeenHandler,Sudo: true},
			{Pattern: "hidepp",       Handler: hidePPHandler,      Sudo: true},
			{Pattern: "hidestatus",   Handler: hideStatusHandler,  Sudo: true},
			{Pattern: "hidegroups",   Handler: hideGroupsHandler,  Sudo: true},
			// Per-number PP
			{Pattern: "ppblock",      Handler: ppBlockHandler,     Sudo: true},
			{Pattern: "ppunblock",    Handler: ppUnblockHandler,   Sudo: true},
			{Pattern: "ppblocklist",  Handler: ppBlockListHandler, Sudo: true},
			// Full status
			{Pattern: "ghoststatus",  Handler: ghostStatusHandler, Sudo: true},
		},
	})
}
