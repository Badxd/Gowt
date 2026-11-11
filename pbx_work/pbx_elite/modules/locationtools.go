package modules

// Location Tools — send and inspect location messages.
//
// Commands:
//   .sendloc <lat> <lon> [name]    — send a location pin
//   .live <lat> <lon> [seconds]    — send a live location (default 60s)
//   .getloc                        — extract lat/lon from quoted location message
//   .myloc                         — instructions to share your live location
//   .locinfo                       — same as .getloc (alias)

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"PbxGb/client"
)

// ── SEND LOCATION ─────────────────────────────────────────────────────────────
func sendLocHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		return ReplyErr(v, "Usage: .sendloc <latitude> <longitude> [place name]\nExample: .sendloc 28.6139 77.2090 New Delhi")
	}

	lat, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return ReplyErr(v, "❌ Invalid latitude. Use decimal format, e.g. 28.6139")
	}
	lon, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return ReplyErr(v, "❌ Invalid longitude. Use decimal format, e.g. 77.2090")
	}

	name := ""
	if len(args) > 2 {
		name = strings.Join(args[2:], " ")
	}

	wac := client.Get()
	ctx := context.Background()

	msg := &waProto.Message{
		LocationMessage: &waProto.LocationMessage{
			DegreesLatitude:  proto.Float64(lat),
			DegreesLongitude: proto.Float64(lon),
			Name:             proto.String(name),
			IsLive:           proto.Bool(false),
		},
	}

	_, err = wac.SendMessage(ctx, v.Info.Chat, msg)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed to send location: %v", err))
	}
	return nil
}

// ── SEND LIVE LOCATION ────────────────────────────────────────────────────────
func sendLiveLocHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		return ReplyErr(v, "Usage: .live <latitude> <longitude> [seconds]\nExample: .live 28.6139 77.2090 300")
	}

	lat, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return ReplyErr(v, "❌ Invalid latitude.")
	}
	lon, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return ReplyErr(v, "❌ Invalid longitude.")
	}

	duration := uint32(60)
	if len(args) > 2 {
		if d, err2 := strconv.ParseUint(args[2], 10, 32); err2 == nil && d > 0 {
			duration = uint32(d)
		}
	}

	wac := client.Get()
	ctx := context.Background()

	_, err = wac.SendMessage(ctx, v.Info.Chat, &waProto.Message{
		LiveLocationMessage: &waProto.LiveLocationMessage{
			DegreesLatitude:      proto.Float64(lat),
			DegreesLongitude:     proto.Float64(lon),
			AccuracyInMeters:     proto.Uint32(10),
			SpeedInMps:           proto.Float32(0),
			DegreesClockwiseFromMagneticNorth: proto.Uint32(0),
			SecondsUntilExpiration: proto.Int32(int32(duration)),
			IsLive:               proto.Bool(true),
		},
	})
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Failed: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("📍 Live location sent for %d seconds.", duration))
}

// ── GET LOCATION FROM QUOTED MSG ─────────────────────────────────────────────
func getLocHandler(v *events.Message, args []string) error {
	if v.Message == nil {
		return ReplyErr(v, "❌ Reply to a location message.")
	}

	var locMsg *waProto.LocationMessage
	var liveMsg *waProto.LiveLocationMessage

	// Direct
	if l := v.Message.GetLocationMessage(); l != nil {
		locMsg = l
	} else if l := v.Message.GetLiveLocationMessage(); l != nil {
		liveMsg = l
	} else if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		// Quoted
		ci := ext.GetContextInfo()
		if ci != nil && ci.QuotedMessage != nil {
			if l := ci.QuotedMessage.GetLocationMessage(); l != nil {
				locMsg = l
			} else if l := ci.QuotedMessage.GetLiveLocationMessage(); l != nil {
				liveMsg = l
			}
		}
	}

	if locMsg == nil && liveMsg == nil {
		return ReplyErr(v, "❌ Reply to a location or live location message.")
	}

	if locMsg != nil {
		lat := locMsg.GetDegreesLatitude()
		lon := locMsg.GetDegreesLongitude()
		name := locMsg.GetName()
		addr := locMsg.GetAddress()
		mapsURL := fmt.Sprintf("https://maps.google.com/?q=%f,%f", lat, lon)

		info := fmt.Sprintf("📍 *Location Info*\n\n"+
			"🌐 *Lat:* `%.6f`\n"+
			"🌐 *Lon:* `%.6f`\n",
			lat, lon)
		if name != "" {
			info += fmt.Sprintf("🏷️ *Name:* %s\n", name)
		}
		if addr != "" {
			info += fmt.Sprintf("📮 *Address:* %s\n", addr)
		}
		info += fmt.Sprintf("\n🗺️ %s", mapsURL)
		return ReplyErr(v, info)
	}

	// live
	lat := liveMsg.GetDegreesLatitude()
	lon := liveMsg.GetDegreesLongitude()
	mapsURL := fmt.Sprintf("https://maps.google.com/?q=%f,%f", lat, lon)
	return ReplyErr(v, fmt.Sprintf(
		"📍 *Live Location*\n\n"+
			"🌐 *Lat:* `%.6f`\n"+
			"🌐 *Lon:* `%.6f`\n"+
			"⏳ *Expires in:* %ds\n\n"+
			"🗺️ %s",
		lat, lon, liveMsg.GetSecondsUntilExpiration(), mapsURL,
	))
}

// ── MY LOC INSTRUCTIONS ───────────────────────────────────────────────────────
func myLocHandler(v *events.Message, args []string) error {
	return ReplyErr(v, "📍 *Share Location*\n\n"+
		"To share your live location:\n"+
		"1. Tap 📎 (Attach)\n"+
		"2. Select *Location*\n"+
		"3. Choose *Share Live Location*\n\n"+
		"Or use `.sendloc <lat> <lon>` to pin a manual location.")
}

func init() {
	Register(ModuleInfo{
		Name:        "LocationTools",
		Description: "Send location pins, live location, extract location from messages",
		Commands: []CommandInfo{
			{Pattern: "sendloc",  Handler: sendLocHandler,     Sudo: true},
			{Pattern: "loc",      Handler: sendLocHandler,     Sudo: true},
			{Pattern: "live",     Handler: sendLiveLocHandler, Sudo: true},
			{Pattern: "liveloc",  Handler: sendLiveLocHandler, Sudo: true},
			{Pattern: "getloc",   Handler: getLocHandler,      Sudo: true},
			{Pattern: "locinfo",  Handler: getLocHandler,      Sudo: true},
			{Pattern: "myloc",    Handler: myLocHandler,       Sudo: true},
		},
	})
}
