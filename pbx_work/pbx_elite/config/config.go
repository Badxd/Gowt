package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI string
	// OwnerJID — pure phone number e.g. "919646191598"
	// Strip @s.whatsapp.net if user accidentally put full JID
	OwnerJID string
	Prefix   string
	Session  string

	// Media download API
	// Set via env: MEDIA_API_URL, MEDIA_API_KEY, MEDIA_API_KEY2
	MediaAPIURL  string
	MediaAPIKey  string
	MediaAPIKey2 string
}

var C *Config

func Load() {
	_ = godotenv.Load()

	prefix := os.Getenv("PREFIX")
	if prefix == "" {
		prefix = "."
	}

	ownerRaw := mustGet("OWNER_JID")

	// Normalize: strip @server suffix, +, spaces
	ownerClean := ownerRaw
	if idx := strings.Index(ownerRaw, "@"); idx != -1 {
		ownerClean = ownerRaw[:idx]
	}
	ownerClean = strings.TrimPrefix(ownerClean, "+")
	ownerClean = strings.ReplaceAll(ownerClean, " ", "")

	log.Printf("[CONFIG] OWNER_JID raw=%q normalized=%q", ownerRaw, ownerClean)

	C = &Config{
		MongoURI: mustGet("MONGO_URI"),
		OwnerJID: ownerClean,
		Prefix:   prefix,
		Session:  os.Getenv("SESSION"),

		// Media download API (optional — defaults built into mediadownload.go)
		MediaAPIURL:  os.Getenv("MEDIA_API_URL"),
		MediaAPIKey:  os.Getenv("MEDIA_API_KEY"),
		MediaAPIKey2: os.Getenv("MEDIA_API_KEY2"),
	}
}

func mustGet(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %q is not set", key)
	}
	return v
}
