package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
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

	"github.com/gorilla/websocket"
	"go.mau.fi/whatsmeow"
 "go.mau.fi/whatsmeow/store/sqlstore"
 waLog "go.mau.fi/whatsmeow/util/log"
 _ "github.com/mattn/go-sqlite3"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	pairMu      sync.Mutex
	justPaired  atomic.Bool
	welcomeSent atomic.Bool
)

// ── Force IPv4 to fix Termux/Heroku IPv6 hangs (Exact logic from working code) ──
func init() {
	websocket.DefaultDialer.NetDial = func(network, addr string) (net.Conn, error) {
		return net.DialTimeout("tcp4", addr, 10*time.Second)
	}
	net.DefaultResolver.PreferGo = true
	net.DefaultResolver.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{Timeout: 10 * time.Second}
		return d.DialContext(ctx, "udp4", address)
	}
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

// Pair using the same temporary SQLite device-store path as the working WtWeb.
// After successful linking, copy the identity into MongoDB and restart the bot.
// The restart loads the durable MongoDB identity as the one active bot session.
func startPairing(wac *whatsmeow.Client, number string) (string, error) {
 pairMu.Lock()
 defer pairMu.Unlock()
 if wac.Store.ID != nil { return "", web.ErrAlreadyLinked }
 digits := onlyDigits(number)
 if len(digits) < 8 || len(digits) > 15 { return "", errors.New("enter a valid number with country code") }
 if os.Getenv("ALLOW_ANY_NUMBER") != "true" && digits != onlyDigits(config.C.OwnerJID) {
  return "", fmt.Errorf("this number does not match OWNER_JID (+%s)", onlyDigits(config.C.OwnerJID))
 }
 if !pairActive.CompareAndSwap(false, true) { return "", errors.New("pairing already in progress") }
 dbFile, err := os.CreateTemp("", "gowt-pair-*.db")
 if err != nil { pairActive.Store(false); return "", err }
 dbPath := dbFile.Name()
 _ = dbFile.Close()
 _ = os.Remove(dbPath)
 container, err := sqlstore.New(context.Background(), "sqlite3", "file:"+dbPath+"?_foreign_keys=on", waLog.Noop)
 if err != nil { pairActive.Store(false); _ = os.Remove(dbPath); return "", fmt.Errorf("pairing store: %w", err) }
 tmp := whatsmeow.NewClient(container.NewDevice(), waLog.Noop)
 var cleanOnce sync.Once
 cleanup := func() { cleanOnce.Do(func() {
  tmp.Disconnect(); _ = container.Close(); _ = os.Remove(dbPath); _ = os.Remove(dbPath+"-wal"); _ = os.Remove(dbPath+"-shm"); pairActive.Store(false)
 }) }
 tmp.AddEventHandler(func(evt interface{}) {
  switch evt.(type) {
  case *events.PairSuccess:
   go func() {
    // Allow WhatsMeow to finish writing device credentials before exporting.
    time.Sleep(2*time.Second)
    encoded, encErr := session.Encode(tmp.Store)
    if encErr != nil { log.Printf("pairing identity export failed: %v", encErr); cleanup(); return }
    if genContainer == nil { log.Print("pairing MongoDB container missing"); cleanup(); return }
    if _, decErr := session.Decode(context.Background(), genContainer, encoded); decErr != nil {
     log.Printf("pairing identity MongoDB import failed: %v", decErr); cleanup(); return
    }
    log.Print("Pairing succeeded: credentials saved to MongoDB; restarting to activate bot")
    cleanup()
    // Heroku restarts the web dyno; on VPS use a process supervisor (systemd).
    os.Exit(0)
   }()
  case *events.PairError:
   log.Print("WhatsApp reported a pairing error")
  }
 })
 if err := tmp.Connect(); err != nil { cleanup(); return "", fmt.Errorf("WhatsApp connect: %w", err) }
 for i:=0; i<30 && !tmp.IsConnected(); i++ { time.Sleep(time.Second) }
 if !tmp.IsConnected() { cleanup(); return "", errors.New("timeout waiting for WhatsApp socket") }
 ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
 defer cancel()
 code, err := tmp.PairPhone(ctx, digits, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
 if err != nil { cleanup(); return "", fmt.Errorf("WhatsApp pair code: %w", err) }
 log.Print("Pair code created; waiting for linked-device confirmation")
 go func() { time.Sleep(3*time.Minute); if pairActive.Load() { cleanup() } }()
 return code, nil
}

var pairActive atomic.Bool

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
		return "", "", fmt.Errorf("session gen not ready")
	}

	digits := onlyDigits(number)
	if len(digits) < 8 || len(digits) > 15 {
		return "", "", errors.New("enter a valid number with country code")
	}

	tmpDevice := genContainer.NewDevice()
	tmpClient := whatsmeow.NewClient(tmpDevice, nil)

	done := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(done) }) }

	jobID := web.NewSessionJob()
	var code string

	tmpClient.AddEventHandler(func(evt interface{}) {
		switch evt.(type) {
		case *events.Connected:
			time.Sleep(2 * time.Second)
			sess, encErr := session.Encode(tmpDevice)
			if encErr != nil {
				log.Printf("⚠️ gensession encode error: %v", encErr)
				web.SetJobStatus(jobID, web.SessionJob{Status: "error", ErrMsg: encErr.Error()})
			} else {
				web.SetJobStatus(jobID, web.SessionJob{Status: "done", Code: code, Session: sess})
				log.Printf("✅ gensession done — session string ready (job %s)", jobID)
			}
			finish()
		case *events.LoggedOut:
			web.SetJobStatus(jobID, web.SessionJob{Status: "error", ErrMsg: "logged out before session was ready"})
			finish()
		}
	})

	if err := tmpClient.Connect(); err != nil {
		tmpClient.Disconnect()
		return "", "", fmt.Errorf("could not reach WhatsApp: %v", err)
	}

	// EXACT LOGIC FROM WORKING CODE
	connected := false
	for i := 0; i < 30; i++ {
		if tmpClient.IsConnected() {
			connected = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !connected {
		tmpClient.Disconnect()
		return "", "", errors.New("timeout waiting for WhatsApp connection")
	}

	log.Println("🔗 gensession: connection ready — requesting pair code")

	var err error
	code, err = tmpClient.PairPhone(context.Background(), digits, true, whatsmeow.PairClientChrome, "Chrome (Windows)")
	if err != nil {
		tmpClient.Disconnect()
		return "", "", fmt.Errorf("could not create pair code: %v", err)
	}

	web.SetJobStatus(jobID, web.SessionJob{Status: "code", Code: code})

	go func() {
		select {
		case <-done:
		case <-time.After(5 * time.Minute):
			web.SetJobStatus(jobID, web.SessionJob{Status: "error", ErrMsg: "timed out waiting for WhatsApp connection"})
		}
		tmpClient.Disconnect()
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
