package modules

import (
	"fmt"
	"math/rand"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

var slapLines = []string{
	"slaps %s around a bit with a large trout 🐟",
	"smacks %s with a wet noodle 🍜",
	"yeeted %s into the shadow realm 💀",
	"gave %s a nice tight slap 👋",
	"hit %s so hard they forgot their name 😤",
}

var insultsHindi = []string{
	"teri maa ki 🖕",
	"bhosdike 😤",
	"madarchod 🤬",
	"bhenchod tu kya samajhta hai apne aap ko 😡",
	"saale ullu ke patthe 🤦",
	"abe gadhe 🐴",
	"tujhe toh teri maa ne bhi nahi maanga tha 💀",
	"lauda lele apna 😂",
	"chutiye ki tarah mat bol 🤡",
	"tu bilkul bekaar hai yaar 😑",
}

var randyLines = []string{
	"bhai kuch nahi aata isko 😤",
	"random moment unlocked 🎲",
	"kya kar raha hai tu seriously 😂",
	"bruh... 💀",
	"yeh toh gaya 😭",
}

func slapHandler(v *events.Message, args []string) error {
	target := ""
	if len(args) > 0 {
		target = strings.Join(args, " ")
	} else if IsReply(v) {
		target = "them"
	} else {
		return ReplyErr(v, "Usage: .slap @user or reply to someone")
	}
	line := slapLines[rand.Intn(len(slapLines))]
	return ReplyErr(v, fmt.Sprintf(line, target))
}

func habuseHandler(v *events.Message, args []string) error {
	insult := insultsHindi[rand.Intn(len(insultsHindi))]
	if IsReply(v) {
		return ReplyErr(v, "Yaar sun... "+insult)
	}
	return ReplyErr(v, insult)
}

func randiHandler(v *events.Message, args []string) error {
	line := randyLines[rand.Intn(len(randyLines))]
	return ReplyErr(v, line)
}

func noobHandler(v *events.Message, args []string) error {
	msg := "Confirmed noob 🤡\n\n"
	msg += "🧠 IQ: -69\n"
	msg += "🎮 Rank: Bronze V\n"
	msg += "💀 K/D: 0.1\n"
	msg += "😭 Win Rate: 0%\n"
	msg += "🏆 Trophies: -10"
	return ReplyErr(v, msg)
}

func chuHandler(v *events.Message, args []string) error {
	lines := []string{
		"chu bhai chu 😂",
		"bilkul pagal hai yeh 💀",
		"chu scene 🤡",
		"bhai isko chu bolte hain 🙏",
	}
	return ReplyErr(v, lines[rand.Intn(len(lines))])
}

func fukHandler(v *events.Message, args []string) error {
	return ReplyErr(v, "🖕😤🖕\n\nBhai khud ko sambhal pehle.")
}

func init() {
	Register(ModuleInfo{
		Name:        "FunExtra",
		Description: "Extra fun and abuse commands",
		Commands: []CommandInfo{
			{Pattern: "slap", Handler: slapHandler, Sudo: true},
			{Pattern: "habuse", Handler: habuseHandler, Sudo: true},
			{Pattern: "randi", Handler: randiHandler, Sudo: true},
			{Pattern: "noob", Handler: noobHandler, Sudo: true},
			{Pattern: "chu", Handler: chuHandler, Sudo: true},
			{Pattern: "fuk", Handler: fukHandler, Sudo: true},
		},
	})
}
