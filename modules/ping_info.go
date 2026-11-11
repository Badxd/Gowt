package modules

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

var botStartTime = time.Now()

// .uptime — how long the bot has been running
func uptimeHandler(v *events.Message, args []string) error {
	return ReplyErr(v, "⏰ *Uptime:* "+formatDuration(time.Since(botStartTime)))
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dh %dm %ds", h, m, s)
}

// .sysinfo — system information
func sysinfoHandler(v *events.Message, args []string) error {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	alloc := float64(mem.Alloc) / 1024 / 1024
	sys := float64(mem.Sys) / 1024 / 1024

	info := fmt.Sprintf(`⚙️ *System Info*
━━━━━━━━━━━━━━━━━━
🖥️ OS: %s / %s
🔢 Go version: %s
⚡ Goroutines: %d
💾 Heap alloc: %.2f MB
📦 Sys memory: %.2f MB
⏰ Uptime: %s`,
		runtime.GOOS, runtime.GOARCH,
		runtime.Version(),
		runtime.NumGoroutine(),
		alloc,
		sys,
		formatDuration(time.Since(botStartTime)),
	)
	return ReplyErr(v, info)
}

// .ip — get public IP of the bot server
func ipHandler(v *events.Message, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org?format=text", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ReplyErr(v, "❌ Could not fetch IP: "+err.Error())
	}
	defer resp.Body.Close()

	var sb strings.Builder
	buf := make([]byte, 64)
	n, _ := resp.Body.Read(buf)
	sb.Write(buf[:n])

	return ReplyErr(v, "🌐 *Server IP:* "+strings.TrimSpace(sb.String()))
}

func init() {
	Register(ModuleInfo{
		Name:        "PingInfo",
		Description: "Bot uptime, sysinfo, public IP",
		Commands: []CommandInfo{
			{Pattern: "uptime", Handler: uptimeHandler, Sudo: true},
			{Pattern: "sysinfo", Handler: sysinfoHandler, Sudo: true},
			{Pattern: "ip", Handler: ipHandler, Sudo: true},
		},
	})
}
