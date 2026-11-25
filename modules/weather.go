package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

type wttrResponse struct {
	CurrentCondition []struct {
		TempC       string `json:"temp_C"`
		TempF       string `json:"temp_F"`
		FeelsLikeC  string `json:"FeelsLikeC"`
		FeelsLikeF  string `json:"FeelsLikeF"`
		Humidity    string `json:"humidity"`
		WindspeedKmph string `json:"windspeedKmph"`
		WeatherDesc []struct {
			Value string `json:"value"`
		} `json:"weatherDesc"`
	} `json:"current_condition"`
	NearestArea []struct {
		AreaName []struct {
			Value string `json:"value"`
		} `json:"areaName"`
		Country []struct {
			Value string `json:"value"`
		} `json:"country"`
	} `json:"nearest_area"`
}

func weatherHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .weather <city name>")
	}
	city := strings.Join(args, "+")
	apiURL := fmt.Sprintf("https://wttr.in/%s?format=j1", url.PathEscape(city))

	resp, err := http.Get(apiURL)
	if err != nil {
		return ReplyErr(v, "❌ Failed to fetch weather: "+err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ReplyErr(v, fmt.Sprintf("❌ Weather API error: %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ReplyErr(v, "❌ Read error")
	}

	var data wttrResponse
	if err := json.Unmarshal(body, &data); err != nil || len(data.CurrentCondition) == 0 {
		return ReplyErr(v, "❌ Could not parse weather data.")
	}

	cur := data.CurrentCondition[0]
	desc := "N/A"
	if len(cur.WeatherDesc) > 0 {
		desc = cur.WeatherDesc[0].Value
	}

	location := city
	if len(data.NearestArea) > 0 {
		a := data.NearestArea[0]
		if len(a.AreaName) > 0 && len(a.Country) > 0 {
			location = a.AreaName[0].Value + ", " + a.Country[0].Value
		}
	}

	msg := fmt.Sprintf(`🌤️ *Weather in %s*

🌡️ Temp: %s°C / %s°F
🤔 Feels like: %s°C / %s°F
💧 Humidity: %s%%
💨 Wind: %s km/h
☁️ Condition: %s`,
		location,
		cur.TempC, cur.TempF,
		cur.FeelsLikeC, cur.FeelsLikeF,
		cur.Humidity,
		cur.WindspeedKmph,
		desc,
	)

	return ReplyErr(v, msg)
}

func init() {
	Register(ModuleInfo{
		Name:        "Weather",
		Description: "Get current weather for a city",
		Commands: []CommandInfo{
			{Pattern: "weather", Handler: weatherHandler, Sudo: true},
		},
	})
}
