package modules

import (
	"fmt"
	"math/rand"

	"go.mau.fi/whatsmeow/types/events"
)

var jokes = []string{
	"Teacher: Why were you late to school?\nStudent: You always say never be early, so I practiced! 😂",
	"A man went to the doctor...\nMan: Doctor, I can't sleep.\nDoctor: Wait till you hear my bill. 😂",
	"Wife: Do you love me?\nHusband: More than Google Maps — I never say you're wrong. 😂",
	"Student: Sir, nothing came from what I studied!\nTeacher: When exactly did you study? 😂",
	"A man asked the shopkeeper: How much is this watch?\nShopkeeper: $500.\nMan: What about if it's broken? 😂",
	"Mom: Wake up, it's morning!\nSon: Why?\nMom: School is today.\nSon: Mom... I'm the teacher. 😂",
	"Wife: Why do you always argue with me?\nHusband: I'm practicing — it'll be useful later. 😂",
}

var quotes = []string{
	"💡 *\"Dreams are not those which come while sleeping, but those which do not let you sleep.\"* — A.P.J. Abdul Kalam",
	"🔥 *\"Difficulties do not break a person — they build them stronger.\"*",
	"⚡ *\"Do tomorrow's work today, and today's work right now.\"* — Kabir",
	"🌟 *\"You only fail when you stop trying.\"*",
	"💪 *\"Nothing in life is easy, but nothing in life is impossible either.\"*",
	"🎯 *\"Success is not what you have — it is what you do.\"*",
	"🌙 *\"Do not fear the darkness of night — it is what brings the light of morning.\"*",
}

var facts = []string{
	"🧠 *Fun Fact:* An octopus has 3 hearts!",
	"🌍 *Fun Fact:* Antarctica is the largest desert on Earth!",
	"🍯 *Fun Fact:* Honey never spoils — edible honey has been found in 3000-year-old Egyptian tombs!",
	"💤 *Fun Fact:* The average person spends about 26 years of their life asleep!",
	"🐘 *Fun Fact:* Elephants are the only animals that cannot jump!",
	"📱 *Fun Fact:* There are 8 billion people in the world but over 15 billion mobile phones!",
	"🌊 *Fun Fact:* Only 5% of the world's oceans have been explored so far!",
	"🦷 *Fun Fact:* Humans ki 2 sets of teeth hoti hain, sharks ki unlimited!",
	"🐟 *Fun Fact:* Goldfish have a memory span of up to 3 months — not 3 seconds!",
	"🌙 *Fun Fact:* Footprints on the Moon can last for millions of years — there is no wind or rain!",
}

func jokeHandler(v *events.Message, args []string) error {
	joke := jokes[rand.Intn(len(jokes))]
	return ReplyErr(v, "😂 *Joke of the Day*\n\n"+joke)
}


func factHandler(v *events.Message, args []string) error {
	fact := facts[rand.Intn(len(facts))]
	return ReplyErr(v, fact)
}

func tossHandler(v *events.Message, args []string) error {
	result := "🪙 *Heads!*"
	if rand.Intn(2) == 0 {
		result = "🪙 *Tails!*"
	}
	return ReplyErr(v, result)
}

func diceHandler(v *events.Message, args []string) error {
	return ReplyErr(v, fmt.Sprintf("🎲 *Dice rolled:* %d", rand.Intn(6)+1))
}

func eightBallHandler(v *events.Message, args []string) error {
	answers := []string{
		"✅ Haan bilkul!", "✅ Pakka!", "✅ 100% YES!",
		"🤔 Maybe...", "🤔 Not sure about that...",
		"❌ I don't think so...", "❌ Definitely not!", "❌ Think again...",
	}
	return ReplyErr(v, "🎱 *8Ball says:* "+answers[rand.Intn(len(answers))])
}

func rpsHandler(v *events.Message, args []string) error {
	moves := []string{"✊ Rock", "✋ Paper", "✌️ Scissors"}
	botMove := moves[rand.Intn(3)]
	return ReplyErr(v, fmt.Sprintf("🎮 *Rock Paper Scissors*\n\nBot chose: *%s*\n\nTumhara move kya tha? 😄", botMove))
}

func init() {
	Register(ModuleInfo{
		Name:        "Fun",
		Description: "Fun commands — joke, quote, fact, games",
		Commands: []CommandInfo{
			{Pattern: "joke",   Handler: jokeHandler,     Sudo: true},
			{Pattern: "fact",   Handler: factHandler,     Sudo: true},
			{Pattern: "toss",   Handler: tossHandler,     Sudo: true},
			{Pattern: "dice",   Handler: diceHandler,     Sudo: true},
			{Pattern: "8ball",  Handler: eightBallHandler, Sudo: true},
			{Pattern: "rps",    Handler: rpsHandler,      Sudo: true},
		},
	})
}
