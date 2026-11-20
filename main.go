package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"PbxGb/client"
	"PbxGb/config"
	"PbxGb/database"
	"PbxGb/modules"
	"PbxGb/mongostore"
	"PbxGb/session"
	"PbxGb/web"

	"github.com/gorilla/websocket"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// ── Force IPv4 to fix Termux/Heroku IPv6 hangs ───────────────────────────────
func init() {
	websocket.DefaultDialer.NetDial = func(network, addr string) (net.Conn, error) {
		return net.DialTimeout("tcp4", addr, 15*time.Second)
	}
	net.DefaultResolver.PreferGo = true
	net.DefaultResolver.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{Timeout: 10 * time.Second}
		return d.DialContext(ctx, "udp4", address)
	}
}

func main() {
	config.Load()

	if err := database.Connect(config.C.MongoURI); err != nil {
		log.Fatalf("❌ MongoDB error: %v", err)
	}
	defer database.Disconnect()

	// Load PM permit settings into memory now that DB is connected.
	modules.InitPMCache()

	ctx := context.Background()

	// WhatsApp keys / sessions / app-state live in MongoDB (no SQLite), so a
	// Heroku restart never loses the linked session.
	container, err := mongostore.New(ctx, database.DB, nil)
	if err != nil {
		log.Fatalf("❌ Mongo store error: %v", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		log.Fatalf("❌ Device error: %v", err)
	}

	// Optional: import an old SESSION string (only when nothing is stored yet).
	if deviceStore.ID == nil && config.C.Session != "" {
		if _, err := session.Decode(ctx, container, config.C.Session); err != nil {
			log.Printf("⚠️  SESSION env could not be imported (%v) — open the website to link instead", err)
		} else if dev, err := container.GetFirstDevice(ctx); err == nil {
			deviceStore = dev
			log.Println("✅ Session imported from SESSION env")
		}
	}

	// Make the mongostore container available for session-gen temp devices.
	genContainer = container

	wac := whatsmeow.NewClient(deviceStore, nil)
	client.Set(wac)

	wac.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {

		case *events.Message:
			modules.CacheMessage(v)

			modules.HandleAntiDelete(v)
			modules.HandleAntiViewOnce(v)
			modules.HandleAutoVV(v)
			modules.HandleEmojiVV(v)
			modules.HandleAntiLink(v)
			modules.HandleAutoSave(v)
			modules.HandleAutoReact(v)

			modules.HandleAFK(v)
			modules.HandleBombMove(v)
			modules.HandleTTTMove(v)
			modules.HandleGbanRejoin(v)
			modules.HandlePM(v)
			modules.HandleMessage(v)

		case *events.CallOffer:
			modules.HandleCall(v)

		case *events.PairSuccess:
			onPairSuccess(wac, v)

		case *events.Connected:
			go onConnected(wac)

		case *events.LoggedOut:
			go onLoggedOut(wac)

		case *events.Disconnected:
			// Auto-reconnect unless we were deliberately unlinked.
			// onLoggedOut calls os.Exit so we only reach here for dropped connections.
			if wac.Store.ID != nil {
				go reconnect(wac)
			}
		}
	})

	// Website first — Heroku needs the port bound quickly, and the pair page
	// is how the (single) session gets added.
	web.Start(web.Hooks{
		Info:       func() web.Info { return buildInfo(wac) },
		Pair:       func(number string) (string, error) { return startPairing(wac, number) },
		GenSession: startGenSession,
		Logout: func() error {
			return doLogout(wac)
		},
	})

	if wac.Store.ID != nil {
		if err := wac.Connect(); err != nil {
			log.Fatalf("❌ Connect error: %v", err)
		}
		log.Println("✅ PBX Elite connected")
	} else {
		log.Println("🔗 No session yet — open the website and link your WhatsApp number")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("🔌 Disconnecting...")
	wac.Disconnect()
	time.Sleep(200 * time.Millisecond)
}

// reconnect is called in a goroutine when a Disconnected event fires.
// It tries to reconnect indefinitely with exponential back-off (cap 60 s).
func reconnect(wac *whatsmeow.Client) {
	backoff := 3 * time.Second
	for {
		if wac.IsConnected() {
			return // something else already reconnected
		}
		log.Printf("🔄 Reconnecting in %s…", backoff)
		time.Sleep(backoff)

		if err := wac.Connect(); err != nil {
			log.Printf("⚠️  Reconnect failed: %v — will retry", err)
			if backoff < 60*time.Second {
				backoff *= 2
			}
			continue
		}
		log.Println("✅ Reconnected to WhatsApp")
		return
	}
}
