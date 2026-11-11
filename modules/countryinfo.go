package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// .country <name> — country information via restcountries.com

type countryData struct {
	Name struct {
		Common   string `json:"common"`
		Official string `json:"official"`
	} `json:"name"`
	Capital    []string          `json:"capital"`
	Region     string            `json:"region"`
	Subregion  string            `json:"subregion"`
	Population int64             `json:"population"`
	Area       float64           `json:"area"`
	Currencies map[string]struct {
		Name   string `json:"name"`
		Symbol string `json:"symbol"`
	} `json:"currencies"`
	Languages map[string]string `json:"languages"`
	Timezones []string          `json:"timezones"`
	Flags     struct {
		PNG string `json:"png"`
	} `json:"flags"`
	TLD      []string `json:"tld"`
	CCA2     string   `json:"cca2"`
	Landlocked bool   `json:"landlocked"`
}

func fetchCountryInfo(name string) (*countryData, error) {
	endpoint := "https://restcountries.com/v3.1/name/" + url.PathEscape(name) + "?fullText=false"

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("User-Agent", "PBX-Elite/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var countries []countryData
	if err := json.Unmarshal(data, &countries); err != nil {
		return nil, fmt.Errorf("parse error: %s", err)
	}
	if len(countries) == 0 {
		return nil, fmt.Errorf("country not found: %s", name)
	}
	return &countries[0], nil
}

func countryHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .country <country name>\nExample: .country India")
	}

	name := strings.Join(args, " ")
	info, err := fetchCountryInfo(name)
	if err != nil {
		return ReplyErr(v, "❌ "+err.Error())
	}

	// Format currencies
	var currencies []string
	for _, c := range info.Currencies {
		currencies = append(currencies, fmt.Sprintf("%s (%s)", c.Name, c.Symbol))
	}

	// Format languages
	var languages []string
	for _, l := range info.Languages {
		languages = append(languages, l)
	}

	capital := "N/A"
	if len(info.Capital) > 0 {
		capital = strings.Join(info.Capital, ", ")
	}

	timezone := "N/A"
	if len(info.Timezones) > 0 {
		timezone = info.Timezones[0]
	}

	tld := "N/A"
	if len(info.TLD) > 0 {
		tld = strings.Join(info.TLD, ", ")
	}

	landlocked := "No"
	if info.Landlocked {
		landlocked = "Yes"
	}

	msg := fmt.Sprintf(`🌍 *%s*
━━━━━━━━━━━━━━━━━━
🏳️ Official: %s
🗺️ Region: %s, %s
🏙️ Capital: %s
👥 Population: %s
📐 Area: %.0f km²
💱 Currency: %s
🗣️ Languages: %s
⏰ Timezone: %s
🌐 TLD: %s
🏔️ Landlocked: %s
🆔 Code: %s`,
		info.Name.Common,
		info.Name.Official,
		info.Region, info.Subregion,
		capital,
		formatPopulation(info.Population),
		info.Area,
		strings.Join(currencies, ", "),
		strings.Join(languages, ", "),
		timezone,
		tld,
		landlocked,
		info.CCA2,
	)
	return ReplyErr(v, msg)
}

func formatPopulation(n int64) string {
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.2fB", float64(n)/1_000_000_000)
	case n >= 1_000_000:
		return fmt.Sprintf("%.2fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func init() {
	Register(ModuleInfo{
		Name:        "CountryInfo",
		Description: "Get detailed country information",
		Commands: []CommandInfo{
			{Pattern: "country", Handler: countryHandler, Sudo: true},
		},
	})
}
