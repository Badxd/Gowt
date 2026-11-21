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
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log" // Logger lyi import add kitta
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
	time.Sleep(500 * time.Millisecond)

	if err := wac.Connect(); err != nil {
		return "", fmt.Errorf("could not reach WhatsApp: %v", err)
	}

	connected := false
	for i := 0; i < 30; i++ {
		if wac.IsConnected() {
			connected = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !connected {
		wac.Disconnect()
		return "", errors.New("timeout waiting for WhatsApp connection")
	}

	log.Println("🔗 Connection ready, requesting pair code")

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
	// FIX 1: Logger ON kitta hai taaki error dikhe
	tmpClient := whatsmeow.NewClient(tmpDevice, waLog.Stdout("WA_GEN", "DEBUG", true))

	done := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(done) }) }

	jobID := web.NewSessionJob()
	var code string

	tmpClient.AddEventHandler(func(evt interface{}) {
		switch evt.(type) {
		// FIX 2: Connected nahi, PairSuccess da wait karo
		case *events.PairSuccess:
			log.Printf("[GEN] ✅ Pair Success! Extracting session...")
			time.Sleep(2 * time.Second) // Wait for keys to sync
			
			if tmpClient.Store.ID != nil {
				sess, encErr := session.Encode(tmpClient.Store)
				if encErr != nil {
					log.Printf("⚠️ gensession encode error: %v", encErr)
					web.SetJobStatus(jobID, web.SessionJob{Status: "error", ErrMsg: encErr.Error()})
				} else {
					web.SetJobStatus(jobID, web.SessionJob{Status: "done", Code: code, Session: sess})
					log.Printf("✅ gensession done — session string ready (job %s)", jobID)
				}
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	code, err = tmpClient.PairPhone(ctx, digits, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		tmpClient.Disconnect()
		return "", "", fmt.Errorf("could not create pair code: %v", err)
	}

	web.SetJobStatus(jobID, web.SessionJob{Status: "code", Code: code})

	go func() {
		select {
		case <-done:
			// Successfully generated
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
