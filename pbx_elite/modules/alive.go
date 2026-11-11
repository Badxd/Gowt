package modules

import (
	"fmt"
	"runtime"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// .alive  — bot health/status check with fancy banner
// .speed  — alias of speedtest (quick ping)

var startTime = time.Now()

func aliveHandler(v *events.Message, args []string) error {
	uptime := time.Since(startTime).Round(time.Second)

	hours := int(uptime.Hours())
	minutes := int(uptime.Minutes()) % 60
	seconds := int(uptime.Seconds()) % 60

	var uptimeStr string
	if hours > 0 {
		uptimeStr = fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		uptimeStr = fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		uptimeStr = fmt.Sprintf("%ds", seconds)
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memMB := float64(memStats.Alloc) / 1024 / 1024
	goroutines := runtime.NumGoroutine()

	msg := fmt.Sprintf(
		"```\n"+
			"╔══════════════════════════╗\n"+
			"║      PBX ELITE IS ALIVE  ║\n"+
			"╚══════════════════════════╝\n"+
			"```\n"+
			"🟢 *Status:* Online\n"+
			"⏱️ *Uptime:* %s\n"+
			"🧠 *Memory:* %.1f MB\n"+
			"🔄 *Goroutines:* %d\n"+
			"🖥️ *OS:* %s/%s\n"+
			"🐹 *Go:* %s",
		uptimeStr,
		memMB,
		goroutines,
		runtime.GOOS,
		runtime.GOARCH,
		runtime.Version(),
	)

	return ReplyErr(v, msg)
}

func init() {
	Register(ModuleInfo{
		Name:        "Alive",
		Description: "Bot health check and uptime status",
		Commands: []CommandInfo{
			{Pattern: "alive", Handler: aliveHandler, Sudo: true},
		},
	})
}
