package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// .crypto <symbol> — get cryptocurrency price via CoinGecko free API

type cgPriceResponse map[string]map[string]float64

var coinIDs = map[string]string{
	"btc":  "bitcoin",
	"eth":  "ethereum",
	"bnb":  "binancecoin",
	"sol":  "solana",
	"xrp":  "ripple",
	"ada":  "cardano",
	"dot":  "polkadot",
	"doge": "dogecoin",
	"shib": "shiba-inu",
	"ltc":  "litecoin",
	"avax": "avalanche-2",
	"matic": "matic-network",
	"link": "chainlink",
	"uni":  "uniswap",
	"usdt": "tether",
	"usdc": "usd-coin",
	"trx":  "tron",
	"atom": "cosmos",
	"near": "near",
	"ftm":  "fantom",
}

func fetchCryptoPrice(symbol string) (float64, float64, float64, error) {
	id, ok := coinIDs[strings.ToLower(symbol)]
	if !ok {
		id = strings.ToLower(symbol) // try raw symbol as coingecko id
	}

	apiURL := fmt.Sprintf(
		"https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd&include_24hr_change=true&include_market_cap=true",
		id,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	req.Header.Set("User-Agent", "PBX-Elite/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var result cgPriceResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, 0, 0, err
	}

	prices, ok := result[id]
	if !ok || prices == nil {
		return 0, 0, 0, fmt.Errorf("coin '%s' not found", symbol)
	}

	price := prices["usd"]
	change := prices["usd_24h_change"]
	mcap := prices["usd_market_cap"]
	return price, change, mcap, nil
}

func cryptoHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .crypto <symbol>\nExample: .crypto btc\nSupported: BTC, ETH, BNB, SOL, XRP, ADA, DOGE, etc.")
	}

	symbol := strings.ToUpper(args[0])
	price, change24h, mcap, err := fetchCryptoPrice(args[0])
	if err != nil {
		return ReplyErr(v, "❌ Could not fetch price for "+symbol+": "+err.Error())
	}

	changeEmoji := "📈"
	if change24h < 0 {
		changeEmoji = "📉"
	}

	mcapStr := ""
	if mcap > 0 {
		mcapStr = fmt.Sprintf("\n📦 Market Cap: $%s", formatLargeNum(mcap))
	}

	msg := fmt.Sprintf(`💰 *%s Price*
━━━━━━━━━━━━━━━━
💵 Price: $%.4f
%s 24h Change: %.2f%%%s
🕐 %s`,
		symbol,
		price,
		changeEmoji, change24h,
		mcapStr,
		time.Now().Format("15:04 UTC"),
	)
	return ReplyErr(v, msg)
}

func formatLargeNum(n float64) string {
	switch {
	case n >= 1e12:
		return fmt.Sprintf("%.2fT", n/1e12)
	case n >= 1e9:
		return fmt.Sprintf("%.2fB", n/1e9)
	case n >= 1e6:
		return fmt.Sprintf("%.2fM", n/1e6)
	default:
		return fmt.Sprintf("%.2f", n)
	}
}

func init() {
	Register(ModuleInfo{
		Name:        "Crypto",
		Description: "Cryptocurrency price lookup via CoinGecko",
		Commands: []CommandInfo{
			{Pattern: "crypto", Handler: cryptoHandler, Sudo: true},
			{Pattern: "price", Handler: cryptoHandler, Sudo: true},
		},
	})
}
