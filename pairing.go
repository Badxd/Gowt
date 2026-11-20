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
	"PbxGb/mongostore"
	"PbxGb/session"
	"PbxGb/web"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	pairMu      sync.Mutex
	justPaired  atomic.Bool
	welcomeSent atomic.Bool
)

// Keep the system DNS and dialers unchanged. WhatsMeow uses its own
// WebSocket transport; overriding gorilla/websocket.DefaultDialer does not
// configure that transport and forcing udp4 DNS can break some networks.

// pairCodeWithReconnect performs a fresh connection on a dropped socket.
// A connected TCP/WebSocket is not proof that the WhatsApp info query will
// succeed, so every attempt has its own timeout and error reporting.
func pairCodeWithReconnect(c *whatsmeow.Client, digits string) (string, error) {
	var last error
	for attempt := 1; attempt <= 3; attempt++ {
		if !c.IsConnected() {
			c.Disconnect()
			if err := c.Connect(); err != nil {
				last = fmt.Errorf("connect: %w", err)
				log.Printf("[pair] attempt %d: %v", attempt, last)
				time.Sleep(2 * time.Second)
				continue
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		code, err := c.PairPhone(ctx, digits, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
		cancel()
		if err == nil {
			return code, nil
		}
		last = err
		log.Printf("[pair] attempt %d: connected=%v error=%v", attempt, c.IsConnected(), err)
		if attempt < 3 {
			c.Disconnect()
			time.Sleep(2 * time.Second)
		}
	}
	return "", fmt.Errorf("WhatsApp pairing failed after 3 attempts: %w", last)
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

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

	wac.Disconnect()
	log.Println("🔗 Pairing: connecting to WhatsApp")
	return pairCodeWithReconnect(wac, digits)
}

func onPairSuccess(wac *whatsmeow.Client, v *events.PairSuccess) {
	justPaired.Store(true)
	log.Printf("✅ Paired as %s — userbot is activating", v.ID.String())
}

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

func onLoggedOut(wac *whatsmeow.Client) {
	log.Println("⚠️ Logged out from phone — clearing session")
	_ = wac.Store.Delete(context.Background())
	time.Sleep(time.Second)
	os.Exit(0)
}

func doLogout(wac *whatsmeow.Client) error {
	log.Println("⚠️ Logging out...")
	err := wac.Logout(context.Background())
	if err != nil {
		return err
	}
	_ = wac.Store.Delete(context.Background())
	time.Sleep(1 * time.Second)
	os.Exit(0)
	return nil
}

var genContainer *mongostore.Container

func startGenSession(number string) (string, string, error) {
	if genContainer == nil {
		return "", "", errors.New("session generator not ready")
	}
	digits := onlyDigits(number)
	if len(digits) < 8 || len(digits) > 15 {
		return "", "", errors.New("enter a valid number with country code")
	}
	// Do not create simultaneous ephemeral WhatsApp clients for repeated clicks.
	if !pairMu.TryLock() {
		return "", "", errors.New("another pairing is running; please wait")
	}
	tmpDevice := genContainer.NewDevice()
	tmpClient := whatsmeow.NewClient(tmpDevice, nil)
	jobID := web.NewSessionJob()
	var code string
	var done sync.Once
	finished := make(chan struct{})
	finish := func() { done.Do(func() { close(finished) }) }
	tmpClient.AddEventHandler(func(evt interface{}) {
		switch evt.(type) {
		case *events.Connected:
			if tmpDevice.ID == nil {
				return
			}
			sess, err := session.Encode(tmpDevice)
			if err != nil {
				web.SetJobStatus(jobID, web.SessionJob{Status: "error", ErrMsg: err.Error()})
			} else {
				web.SetJobStatus(jobID, web.SessionJob{Status: "done", Code: code, Session: sess})
				log.Printf("✅ gensession ready (job %s)", jobID)
			}
			finish()
		case *events.LoggedOut:
			web.SetJobStatus(jobID, web.SessionJob{Status: "error", ErrMsg: "logged out before session was ready"})
			finish()
		case *events.PairError:
			log.Printf("⚠️ gensession pair error (job %s)", jobID)
		}
	})
	var err error
	code, err = pairCodeWithReconnect(tmpClient, digits)
	if err != nil {
		tmpClient.Disconnect()
		pairMu.Unlock()
		return "", "", err
	}
	web.SetJobStatus(jobID, web.SessionJob{Status: "code", Code: code})
	go func() {
		defer pairMu.Unlock()
		defer tmpClient.Disconnect()
		select {
		case <-finished:
		case <-time.After(5 * time.Minute):
			web.SetJobStatus(jobID, web.SessionJob{Status: "error", ErrMsg: "timed out waiting for WhatsApp login"})
		}
	}()
	return jobID, code, nil
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
