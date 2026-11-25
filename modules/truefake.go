package modules

import (
	"fmt"
	"math/rand"

	"go.mau.fi/whatsmeow/types/events"
)

func trueOrFakeHandler(v *events.Message, args []string) error {
	choices := []string{"✅ TRUE", "❌ FAKE", "🤔 MAYBE", "💯 100% TRUE", "🗑️ Pure BS"}
	return ReplyErr(v, fmt.Sprintf("🎲 Verdict: *%s*", choices[rand.Intn(len(choices))]))
}

func rateHandler(v *events.Message, args []string) error {
	rating := rand.Intn(11) // 0-10
	bar := ""
	for i := 0; i < rating; i++ {
		bar += "⭐"
	}
	if bar == "" {
		bar = "💩"
	}
	return ReplyErr(v, fmt.Sprintf("📊 Rating: *%d/10*\n%s", rating, bar))
}

func shipHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		return ReplyErr(v, "Usage: .ship <name1> <name2>")
	}
	name1 := args[0]
	name2 := args[1]
	pct := rand.Intn(101)

	bar := ""
	filled := pct / 10
	for i := 0; i < 10; i++ {
		if i < filled {
			bar += "❤️"
		} else {
			bar += "🖤"
		}
	}

	var verdict string
	switch {
	case pct >= 90:
		verdict = "💍 SOULMATES!"
	case pct >= 70:
		verdict = "💕 Great match!"
	case pct >= 50:
		verdict = "😊 Decent couple"
	case pct >= 30:
		verdict = "😬 Needs work..."
	default:
		verdict = "💔 Not meant to be"
	}

	return ReplyErr(v, fmt.Sprintf("💏 *%s* ❤️ *%s*\n\n%s %d%%\n\n%s", name1, name2, bar, pct, verdict))
}

func gayRateHandler(v *events.Message, args []string) error {
	pct := rand.Intn(101)
	return ReplyErr(v, fmt.Sprintf("🏳️‍🌈 Gay rate: *%d%%* 💅", pct))
}

func ppSizeHandler(v *events.Message, args []string) error {
	sizes := []string{"🍆", "🍆🍆", "🍆🍆🍆", "8=D", "8==D", "8===D", "8====D"}
	return ReplyErr(v, fmt.Sprintf("📏 PP Size: *%s*", sizes[rand.Intn(len(sizes))]))
}

func init() {
	Register(ModuleInfo{
		Name:        "FunGames",
		Description: "Fun rating/game commands: ship, rate, truefake",
		Commands: []CommandInfo{
			{Pattern: "truefake", Handler: trueOrFakeHandler, Sudo: true},
			{Pattern: "rate", Handler: rateHandler, Sudo: true},
			{Pattern: "ship", Handler: shipHandler, Sudo: true},
			{Pattern: "gayrate", Handler: gayRateHandler, Sudo: true},
			{Pattern: "pp", Handler: ppSizeHandler, Sudo: true},
		},
	})
}
