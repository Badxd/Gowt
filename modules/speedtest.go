package modules

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

func speedtestHandler(v *events.Message, args []string) error {
	ReplyErr(v, "🚀 Running speedtest, please wait...")

	// Download test: 10MB file from fast.com CDN proxy or Cloudflare
	testURL := "https://speed.cloudflare.com/__down?bytes=10000000"

	start := time.Now()
	resp, err := http.Get(testURL)
	if err != nil {
		return ReplyErr(v, "❌ Speedtest failed: "+err.Error())
	}
	defer resp.Body.Close()

	n, err := io.Copy(io.Discard, resp.Body)
	elapsed := time.Since(start)

	if err != nil && n == 0 {
		return ReplyErr(v, "❌ Download failed: "+err.Error())
	}

	mbits := float64(n*8) / elapsed.Seconds() / 1_000_000
	mbytes := float64(n) / elapsed.Seconds() / 1_000_000
	totalMB := float64(n) / 1_000_000

	result := fmt.Sprintf(`⚡ *Speedtest Result*

📥 *Download:*
• Speed: %.2f Mbps
• Speed: %.2f MB/s
• Downloaded: %.2f MB
• Time: %.2fs

🖥️ *Server:* Cloudflare CDN`, mbits, mbytes, totalMB, elapsed.Seconds())

	return ReplyErr(v, result)
}

func init() {
	Register(ModuleInfo{
		Name:        "Speedtest",
		Description: "Test server download speed",
		Commands: []CommandInfo{
			{Pattern: "speedtest", Handler: speedtestHandler, Sudo: true},
		},
	})
}
