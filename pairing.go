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

	// Register the QR-ready listener BEFORE connecting so we never miss the event.
	qrReady := make(chan struct{}, 1)
	handlerID := wac.AddEventHandler(func(evt interface{}) {
		if _, ok := evt.(*events.QR); ok {
			select {
			case qrReady <- struct{}{}:
			default:
			}
		}
	})
	defer wac.RemoveEventHandler(handlerID)

	if err := wac.Connect(); err != nil {
		return "", fmt.Errorf("could not reach WhatsApp: %v", err)
	}

	// Wait for whatsmeow to fire *events.QR — that signals the noise handshake
	// is done and PairPhone is safe to call. 30-second timeout; if the event
	// doesn't arrive we proceed anyway (PairPhone will return a clear error).
	select {
	case <-qrReady:
		log.Println("🔗 QR event received — connection ready, requesting pair code")
	case <-time.After(30 * time.Second):
		log.Println("⚠️ QR event timeout — attempting PairPhone anyway")
	}

	var (
		code string
		err  error
	)
	for i := 0; i < 5; i++ {
		if i > 0 {
			time.Sleep(3 * time.Second)
		}
		code, err = wac.PairPhone(context.Background(), digits, true, whatsmeow.PairClientChrome, "Chrome (Windows)")
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

// ─── Session-string generation ───────────────────────────────────────────────
// genContainer is set by main.go so pairing.go can create temp devices
// without importing the database package directly.
var genContainer *mongostore.Container

// startGenSession starts a background job that:
//  1. Creates a fresh unpaired device (temp, not persisted),
//  2. Connects to WhatsApp and requests a pair code,
//  3. Waits for the Connected event,
//  4. Encodes the session string and stores it on the job.
//
// It returns (jobID, pairCode, error). The browser polls /api/gensession/poll?id=.
func startGenSession(number string) (string, string, error) {
	if genContainer == nil {
		return "", "", fmt.Errorf("session gen not ready")
	}

	digits := onlyDigits(number)
	if len(digits) < 8 || len(digits) > 15 {
		return "", "", errors.New("enter a valid number with country code")
	}

	// Fresh unpaired device — keys generated in-memory, NOT saved to Mongo yet.
	tmpDevice := genContainer.NewDevice()
	tmpClient := whatsmeow.NewClient(tmpDevice, nil)

	// Pre-register the QR-ready listener so we never miss the event.
	// Also register the Connected/LoggedOut handlers here so they're in place
	// before PairPhone is called — avoids a race where the phone scans fast.
	qrReady := make(chan struct{}, 1)
	done := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(done) }) }

	// Pre-register job ID now so the handler can write to it.
	jobID := web.NewSessionJob()

	tmpClient.AddEventHandler(func(evt interface{}) {
		switch evt.(type) {
		case *events.QR:
			select {
			case qrReady <- struct{}{}:
			default:
			}
		case *events.Connected:
			// Small delay — ensure whatsmeow has written Account back to the device.
			time.Sleep(2 * time.Second)
			sess, encErr := session.Encode(tmpDevice)
			if encErr != nil {
				log.Printf("⚠️ gensession encode error: %v", encErr)
				web.SetJobStatus(jobID, web.SessionJob{Status: "error", ErrMsg: encErr.Error()})
			} else {
				// Include Code so SetJobStatus doesn't blank the previously stored code.
				web.SetJobStatus(jobID, web.SessionJob{Status: "done", Code: code, Session: sess})
				log.Printf("✅ gensession done — session string ready (job %s)", jobID)
			}
			finish()
		case *events.LoggedOut:
			web.SetJobStatus(jobID, web.SessionJob{Status: "error", ErrMsg: "logged out before session was ready"})
			finish()
		}
	})

	// Connect to WhatsApp.
	if err := tmpClient.Connect(); err != nil {
		tmpClient.Disconnect()
		return "", "", fmt.Errorf("could not reach WhatsApp: %v", err)
	}

	// Wait for whatsmeow to fire *events.QR — that signals the noise handshake
	// is complete and PairPhone is safe. 30-second timeout.
	select {
	case <-qrReady:
		log.Println("🔗 gensession: QR event received — requesting pair code")
	case <-time.After(30 * time.Second):
		log.Println("⚠️ gensession: QR event timeout — attempting PairPhone anyway")
	}

	// Request pair code.
	code, err := tmpClient.PairPhone(context.Background(), digits, true, whatsmeow.PairClientChrome, "Chrome (Windows)")
	if err != nil {
		tmpClient.Disconnect()
		return "", "", fmt.Errorf("could not create pair code: %v", err)
	}

	// Store code on the job so the poll endpoint can serve it immediately.
	web.SetJobStatus(jobID, web.SessionJob{Status: "code", Code: code})

	// Background goroutine holds the client alive until Connected or timeout.
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
