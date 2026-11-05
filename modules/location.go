package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"


	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// .location <place>  — send a location pin for a place name
// .coords <lat> <lon> — send a location pin for exact coordinates
// .weather2 <place>  — already in weather.go, skip

type nominatimResult struct {
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
}

func geocode(place string) (float64, float64, string, error) {
	apiURL := "https://nominatim.openstreetmap.org/search?q=" +
		url.QueryEscape(place) + "&format=json&limit=1"

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return 0, 0, "", err
	}
	req.Header.Set("User-Agent", "PBX-Elite-Bot/1.0")

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, 0, "", fmt.Errorf("geocode request failed: %v", err)
	}
	defer resp.Body.Close()

	var results []nominatimResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return 0, 0, "", fmt.Errorf("geocode decode failed: %v", err)
	}

	if len(results) == 0 {
		return 0, 0, "", fmt.Errorf("no results for: %s", place)
	}

	lat, _ := strconv.ParseFloat(results[0].Lat, 64)
	lon, _ := strconv.ParseFloat(results[0].Lon, 64)
	return lat, lon, results[0].DisplayName, nil
}

func sendLocationPin(v *events.Message, lat, lon float64, name, address string) error {
	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	latF32 := float32(lat)
	lonF32 := float32(lon)

	msg := &waE2E.Message{
		LocationMessage: &waE2E.LocationMessage{
			DegreesLatitude:  &latF32,
			DegreesLongitude: &lonF32,
			Name:             &name,
			Address:          &address,
		},
	}

	_, err := wac.SendMessage(ctx, v.Info.Chat, msg)
	return err
}

func locationHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .location <place name>\nExample: .location Eiffel Tower Paris")
	}

	place := strings.Join(args, " ")
	ReplyErr(v, fmt.Sprintf("🔍 Searching for: *%s*...", place))

	lat, lon, display, err := geocode(place)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Location not found: %v", err))
	}

	// short name = first part of display
	parts := strings.SplitN(display, ",", 2)
	name := strings.TrimSpace(parts[0])

	return sendLocationPin(v, lat, lon, name, display)
}

func coordsHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		return ReplyErr(v, "Usage: .coords <latitude> <longitude>\nExample: .coords 48.8584 2.2945")
	}

	lat, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return ReplyErr(v, "❌ Invalid latitude.")
	}
	lon, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return ReplyErr(v, "❌ Invalid longitude.")
	}

	label := ""
	if len(args) > 2 {
		label = strings.Join(args[2:], " ")
	} else {
		label = fmt.Sprintf("%.4f, %.4f", lat, lon)
	}

	return sendLocationPin(v, lat, lon, label, label)
}

func fakeLiveLocationHandler(v *events.Message, args []string) error {
	// WhatsApp live location requires periodic updates — instead, send static pin
	// with a label indicating "live" to mimic the look
	if len(args) < 2 {
		return ReplyErr(v, "Usage: .liveloc <lat> <lon> [label]")
	}

	lat, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return ReplyErr(v, "❌ Invalid latitude.")
	}
	lon, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return ReplyErr(v, "❌ Invalid longitude.")
	}

	label := "📍 Live Location"
	if len(args) > 2 {
		label = strings.Join(args[2:], " ")
	}

	wac := getClientSafe()
	if wac == nil {
		return ReplyErr(v, "❌ Client not available.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	latF32 := float32(lat)
	lonF32 := float32(lon)
	accuracy := float32(10.0) // metres
	speed := float32(0.0)
	bearing := float32(0.0)

	msg := &waE2E.Message{
		LiveLocationMessage: &waE2E.LiveLocationMessage{
			DegreesLatitude:        &latF32,
			DegreesLongitude:       &lonF32,
			AccuracyInMeters:       &accuracy,
			SpeedInMps:             &speed,
			DegreesClockwiseFromMagneticNorth: &bearing,
			Caption:                &label,
		},
	}

	_, err = wac.SendMessage(ctx, v.Info.Chat, msg)
	return err
}

func init() {
	Register(ModuleInfo{
		Name:        "Location",
		Description: "Send location pins and coordinates",
		Commands: []CommandInfo{
			{Pattern: "location", Handler: locationHandler, Sudo: true},
			{Pattern: "loc", Handler: locationHandler, Sudo: true},
			{Pattern: "coords", Handler: coordsHandler, Sudo: true},
			{Pattern: "liveloc", Handler: fakeLiveLocationHandler, Sudo: true},
		},
	})
}
