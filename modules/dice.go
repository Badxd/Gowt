package modules

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// .dice [NdM]  — roll dice, e.g. .dice 2d6  (default 1d6)
// .flip        — flip a coin
// .rng <min> <max>  — random number between min and max
// .pick <item1> | <item2> | ...  — pick random item from list
// .8ball <question>  — magic 8-ball answer

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

var eightBallAnswers = []string{
	"✅ It is certain.",
	"✅ It is decidedly so.",
	"✅ Without a doubt.",
	"✅ Yes, definitely.",
	"✅ You may rely on it.",
	"✅ As I see it, yes.",
	"✅ Most likely.",
	"✅ Outlook good.",
	"✅ Yes.",
	"✅ Signs point to yes.",
	"🤔 Reply hazy, try again.",
	"🤔 Ask again later.",
	"🤔 Better not tell you now.",
	"🤔 Cannot predict now.",
	"🤔 Concentrate and ask again.",
	"❌ Don't count on it.",
	"❌ My reply is no.",
	"❌ My sources say no.",
	"❌ Outlook not so good.",
	"❌ Very doubtful.",
}


func flipHandler(v *events.Message, args []string) error {
	if r.Intn(2) == 0 {
		return ReplyErr(v, "🪙 *Heads!*")
	}
	return ReplyErr(v, "🪙 *Tails!*")
}

func rngHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		return ReplyErr(v, "Usage: .rng <min> <max>\nExample: .rng 1 100")
	}
	min, err1 := strconv.Atoi(args[0])
	max, err2 := strconv.Atoi(args[1])
	if err1 != nil || err2 != nil {
		return ReplyErr(v, "❌ Please provide valid integers")
	}
	if min >= max {
		return ReplyErr(v, "❌ Min must be less than max")
	}
	result := r.Intn(max-min+1) + min
	return ReplyErr(v, fmt.Sprintf("🎰 Random number between *%d* and *%d*: *%d*", min, max, result))
}

func pickHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .pick item1 | item2 | item3\nExample: .pick pizza | burger | sushi")
	}

	full := strings.Join(args, " ")
	items := strings.Split(full, "|")

	var cleaned []string
	for _, item := range items {
		s := strings.TrimSpace(item)
		if s != "" {
			cleaned = append(cleaned, s)
		}
	}

	if len(cleaned) < 2 {
		return ReplyErr(v, "❌ Provide at least 2 options separated by |")
	}

	chosen := cleaned[r.Intn(len(cleaned))]
	return ReplyErr(v, "🎯 *Picked:* "+chosen)
}


func init() {
	Register(ModuleInfo{
		Name:        "Random",
		Description: "Dice, coin flip, random number, pick, 8ball",
		Commands: []CommandInfo{
			{Pattern: "flip", Handler: flipHandler, Sudo: true},
			{Pattern: "rng", Handler: rngHandler, Sudo: true},
			{Pattern: "pick", Handler: pickHandler, Sudo: true},
		},
	})
}
