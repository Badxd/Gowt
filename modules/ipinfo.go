package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// .ipinfo <IP or domain>  — get geolocation + details for an IP/domain
// Uses ip-api.com (free, no key, 45 req/min limit)

type ipAPIResponse struct {
	Status      string  `json:"status"`
	Message     string  `json:"message"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"regionName"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	AS          string  `json:"as"`
	Query       string  `json:"query"`
}

func fetchIPInfo(ipOrDomain string) (*ipAPIResponse, error) {
	// Resolve domain to IP if needed
	target := ipOrDomain
	if net.ParseIP(ipOrDomain) == nil {
		// It's a domain — resolve it
		addrs, err := net.LookupHost(ipOrDomain)
		if err != nil || len(addrs) == 0 {
			return nil, fmt.Errorf("could not resolve domain: %s", ipOrDomain)
		}
		target = addrs[0]
	}

	endpoint := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,message,country,countryCode,regionName,city,zip,lat,lon,timezone,isp,org,as,query", target)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IP API returned %d", resp.StatusCode)
	}

	data, _ := io.ReadAll(resp.Body)
	var result ipAPIResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	if result.Status == "fail" {
		return nil, fmt.Errorf("lookup failed: %s", result.Message)
	}
	return &result, nil
}

func ipinfoHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .ipinfo <IP or domain>\nExample: .ipinfo 8.8.8.8\nExample: .ipinfo google.com")
	}

	query := strings.TrimSpace(args[0])
	// Remove protocol prefix if present
	query = strings.TrimPrefix(query, "https://")
	query = strings.TrimPrefix(query, "http://")
	// Remove path
	if idx := strings.Index(query, "/"); idx != -1 {
		query = query[:idx]
	}

	info, err := fetchIPInfo(query)
	if err != nil {
		return ReplyErr(v, "❌ "+err.Error())
	}

	out := fmt.Sprintf(`🌐 *IP Information*

🔍 *Query:* %s
🏳️ *Country:* %s (%s)
🏙️ *City:* %s, %s
📮 *ZIP:* %s
📍 *Coordinates:* %.4f, %.4f
⏰ *Timezone:* %s
📡 *ISP:* %s
🏢 *Organization:* %s
🔗 *AS:* %s
🗺️ *Maps:* https://maps.google.com/?q=%.4f,%.4f`,
		info.Query,
		info.Country, info.CountryCode,
		info.City, info.Region,
		info.Zip,
		info.Lat, info.Lon,
		info.Timezone,
		info.ISP,
		info.Org,
		info.AS,
		info.Lat, info.Lon,
	)

	return ReplyErr(v, out)
}

func init() {
	Register(ModuleInfo{
		Name:        "IPInfo",
		Description: "Get geolocation and network info for an IP or domain",
		Commands: []CommandInfo{
			{Pattern: "ipinfo", Handler: ipinfoHandler, Sudo: true},
		},
	})
}
