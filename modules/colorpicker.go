package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// .color <#hex or rgb(r,g,b) or name>
// Uses The Color API (free, no key): https://www.thecolorapi.com/

type colorAPIResponse struct {
	Hex struct {
		Value string `json:"value"`
		Clean string `json:"clean"`
	} `json:"hex"`
	RGB struct {
		R int `json:"r"`
		G int `json:"g"`
		B int `json:"b"`
	} `json:"rgb"`
	HSL struct {
		H int `json:"h"`
		S int `json:"s"`
		L int `json:"l"`
	} `json:"hsl"`
	Name struct {
		Value string `json:"value"`
	} `json:"name"`
	Cmyk struct {
		C int `json:"c"`
		M int `json:"m"`
		Y int `json:"y"`
		K int `json:"k"`
	} `json:"cmyk"`
	Image struct {
		Named string `json:"named"`
		Bare  string `json:"bare"`
	} `json:"image"`
	Contrast struct {
		Value string `json:"value"`
	} `json:"contrast"`
}

var hexRegex = regexp.MustCompile(`^#?([0-9a-fA-F]{6}|[0-9a-fA-F]{3})$`)
var rgbRegex = regexp.MustCompile(`^rgb\s*\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*\)$`)

func fetchColorInfo(query string) (*colorAPIResponse, error) {
	var endpoint string
	query = strings.TrimSpace(query)

	if hexRegex.MatchString(query) {
		clean := strings.TrimPrefix(query, "#")
		endpoint = "https://www.thecolorapi.com/id?hex=" + clean
	} else if rgbRegex.MatchString(strings.ToLower(query)) {
		m := rgbRegex.FindStringSubmatch(strings.ToLower(query))
		endpoint = fmt.Sprintf("https://www.thecolorapi.com/id?rgb=%s,%s,%s", m[1], m[2], m[3])
	} else {
		// Try as color name — convert to hex via rough lookup
		endpoint = "https://www.thecolorapi.com/id?named=" + strings.ReplaceAll(query, " ", "%20")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("color API returned %d", resp.StatusCode)
	}
	data, _ := io.ReadAll(resp.Body)
	var cr colorAPIResponse
	if err := json.Unmarshal(data, &cr); err != nil {
		return nil, err
	}
	return &cr, nil
}

func hexToRGB(hex string) (r, g, b int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	val, _ := strconv.ParseInt(hex, 16, 32)
	r = int(val >> 16 & 0xFF)
	g = int(val >> 8 & 0xFF)
	b = int(val & 0xFF)
	return
}

func colorHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .color <#hex | rgb(r,g,b) | name>\nExample: .color #FF5733\nExample: .color rgb(255,87,51)\nExample: .color coral")
	}

	query := strings.Join(args, " ")
	cr, err := fetchColorInfo(query)
	if err != nil {
		return ReplyErr(v, "❌ Could not fetch color info: "+err.Error())
	}

	r, g, b := hexToRGB(cr.Hex.Clean)
	_ = r
	_ = g
	_ = b

	out := fmt.Sprintf(`🎨 *Color Info*

*Name:* %s
*HEX:* %s
*RGB:* rgb(%d, %d, %d)
*HSL:* hsl(%d°, %d%%, %d%%)
*CMYK:* cmyk(%d, %d, %d, %d)
*Contrast:* %s

🖼️ Preview: %s`,
		cr.Name.Value,
		cr.Hex.Value,
		cr.RGB.R, cr.RGB.G, cr.RGB.B,
		cr.HSL.H, cr.HSL.S, cr.HSL.L,
		cr.Cmyk.C, cr.Cmyk.M, cr.Cmyk.Y, cr.Cmyk.K,
		cr.Contrast.Value,
		cr.Image.Named,
	)

	return ReplyErr(v, out)
}

func init() {
	Register(ModuleInfo{
		Name:        "ColorPicker",
		Description: "Get color info from hex, RGB, or name",
		Commands: []CommandInfo{
			{Pattern: "color", Handler: colorHandler, Sudo: true},
		},
	})
}
