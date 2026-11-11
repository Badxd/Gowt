package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"PbxGb/client"
	"PbxGb/config"
	"PbxGb/database"
	"PbxGb/modules"
	"PbxGb/mongostore"
	"PbxGb/web"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

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
		if _, err := client.Decode(ctx, container, config.C.Session); err != nil {
			log.Printf("⚠️ SESSION env could not be imported (%v) — open the website to link instead", err)
		} else if dev, err := container.GetFirstDevice(ctx); err == nil {
			deviceStore = dev
			log.Println("✅ Session imported from SESSION env")
		}
	}

	wac := whatsmeow.NewClient(deviceStore, nil)
	client.Set(wac)

	wac.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {

		case *events.Message:
			// Cache every message for anti-delete
			modules.CacheMessage(v)

			modules.HandleAntiDelete(v)
			modules.HandleAntiViewOnce(v)
			modules.HandleAutoVV(v)
			modules.HandleEmojiVV(v)
			modules.HandleAntiLink(v)
			modules.HandleAutoSave(v)
			modules.HandleAutoReact(v)

			// AFK auto-reply / auto-clear
			modules.HandleAFK(v)

			// Games
			modules.HandleBombMove(v)
			modules.HandleTTTMove(v)

			// PM Permit
			modules.HandleGbanRejoin(v)
			modules.HandlePM(v)

			// Command handler (all .commands)
			modules.HandleMessage(v)

		case *events.CallOffer:
			modules.HandleCall(v)
			modules.TrackCallOffer(v)

		case *events.CallTerminate:
			modules.TrackCallTerminate(v)

		case *events.PairSuccess:
			onPairSuccess(wac, v)

		case *events.Connected:
			go onConnected(wac)

		case *events.LoggedOut:
			go onLoggedOut(wac)
		}
	})

	// Website first — Heroku needs the port bound quickly, and the pair page
	// is how the (single) session gets added.
	web.Start(web.Hooks{
		Info: func() web.Info { return buildInfo(wac) },
		Pair: func(number string) (string, error) { return startPairing(wac, number) },
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
