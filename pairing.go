package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"PbxGb/config"
	"PbxGb/modules"
	"PbxGb/web"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	pairMu       sync.Mutex
	justPaired   atomic.Bool
	welcomeSent  atomic.Bool
)

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// startPairing creates the pair code for the website. Only ONE session is allowed:
// once the client has an ID the endpoint is closed.
func startPairing(wac *whatsmeow.Client, number string) (string, error) {
	pairMu.Lock()
	defer pairMu.Unlock()

	if wac.Store.ID != nil {
		return "", web.ErrAlreadyLinked
	}

	digits := onlyDigits(number)
	if len(digits) < 8 || len(digits) > 15 {
		return "", errors.New("enter a valid number with country code")
	}
	if os.Getenv("ALLOW_ANY_NUMBER") != "true" && digits != onlyDigits(config.C.OwnerJID) {
		return "", fmt.Errorf("this number does not match OWNER_JID (+%s)", onlyDigits(config.C.OwnerJID))
	}

	// always start from a fresh, unpaired websocket
	wac.Disconnect()
	time.Sleep(500 * time.Millisecond)
	if err := wac.Connect(); err != nil {
		return "", fmt.Errorf("could not reach WhatsApp: %v", err)
	}

	var (
		code string
		err  error
	)
	for i := 0; i < 5; i++ {
		time.Sleep(2 * time.Second)
		code, err = wac.PairPhone(context.Background(), digits, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
		if err == nil {
			return code, nil
		}
		log.Printf("⚠️ pair attempt %d failed: %v", i+1, err)
	}
	return "", fmt.Errorf("could not create a pair code: %v", err)
}

func onPairSuccess(wac *whatsmeow.Client, v *events.PairSuccess) {
	justPaired.Store(true)
	log.Printf("✅ Paired as %s — userbot is activating", v.ID.String())
}

// onConnected — right after the first pairing the userbot announces itself in
// the owner's own chat. (The session itself is already stored in MongoDB by the
// Mongo store, nothing else to save.)
func onConnected(wac *whatsmeow.Client) {
	if wac.Store.ID == nil {
		return
	}
	if justPaired.Load() && welcomeSent.CompareAndSwap(false, true) {
		time.Sleep(4 * time.Second)
		text := fmt.Sprintf(
			"┌───⭓ %s ⭓───\n│ ✅ Userbot activated on this number\n│ 📌 Type *%smenu* to see all commands\n│ 🏓 Type *%sping* to test\n│\n│ 📢 %s\n│ 💬 %s\n│ 👑 %s\n└───────────────────⭓",
			modules.BotNameFancy, config.C.Prefix, config.C.Prefix,
			modules.UpdateURL, modules.ChatURL, modules.OwnerURL,
		)
		_, _ = wac.SendMessage(context.Background(), wac.Store.ID.ToNonAD(), &waProto.Message{Conversation: &text})
	}
}

// onLoggedOut — the phone removed this linked device. Wipe everything so the
// website offers the pair page again, then exit (Heroku restarts the dyno).
func onLoggedOut(wac *whatsmeow.Client) {
	log.Println("⚠️ Logged out from phone — clearing session")
	_ = wac.Store.Delete(context.Background())
	time.Sleep(time.Second)
	os.Exit(0)
}

func buildInfo(wac *whatsmeow.Client) web.Info {
	up := time.Since(modules.StartTime)
	info := web.Info{
		BotName:   modules.BotName,
		Version:   modules.BotVersion,
		Prefix:    config.C.Prefix,
		Uptime:    modules.FormatUptime(up),
		UptimeSec: int64(up.Seconds()),
		Commands:  modules.CommandCount(),
	}
	if wac.Store.ID != nil {
		info.Paired = true
		info.Number = wac.Store.ID.User
		info.Name = wac.Store.PushName
		info.Connected = wac.IsConnected() && wac.IsLoggedIn()
		for _, g := range modules.MenuData() {
			info.Groups = append(info.Groups, web.Group{Icon: g.Icon, Title: g.Title, Cmds: g.Cmds})
		}
	}
	return info
}
