package modules

import (
	"context"
	"fmt"
	"strconv"
	"time"


	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// .typing [seconds]  — show "typing..." indicator in current chat (default 3s, max 30s)
// .recording [sec]   — show "recording audio..." indicator (default 3s, max 30s)
// .online            — set presence to Available
// .offline           — set presence to Unavailable





func init() {
	Register(ModuleInfo{
		Name:        "Typing & Presence",
		Description: "Show typing/recording indicators and set online/offline presence",
		Commands: []CommandInfo{
		},
	})
}
