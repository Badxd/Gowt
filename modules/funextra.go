package modules

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"PbxGb/client"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

// ── STATIC LISTS ─────────────────────────────────────────────────────────

var compliments = []string{
	"You're an awesome friend! 💙", "You light up the room! ✨",
	"You're someone's reason to smile! 😊", "You're even better than a unicorn! 🦄",
	"You're a gift to those around you! 🎁", "You're a smart cookie! 🍪",
	"You're awesome! 🌟", "You have the best laugh! 😄",
	"You're gorgeous! 💖", "You're more helpful than you realize! 🤝",
	"You have a great sense of humor! 😂", "You're really something special! ⭐",
	"You're an incredible friend! 🫂", "Your perspective is refreshing! 🌈",
	"You're making a difference! 🌍", "You're stronger than you think! 💪",
	"Your smile is contagious! 😁", "You're one of a kind! 💎",
	"You bring out the best in people! 👏", "You're inspiring! 🌟",
}

var dares = []string{
	"Send a screenshot of your gallery!",
	"Let someone else write a status on your WhatsApp!",
	"Call a random contact and sing them a song!",
	"Post an embarrassing selfie!",
	"Do 20 pushups and send a video!",
	"Change your profile picture to something embarrassing for 24 hours!",
	"Send a voice note singing the alphabet!",
	"Let the group choose your status for a day!",
	"Tell the group your most embarrassing moment!",
	"Share your last 5 Google searches!",
	"Dance in front of everyone for 1 minute!",
	"Do your best impression of someone in the group!",
	"Speak in an accent for the next 10 minutes!",
	"Let someone go through your phone for 2 minutes!",
	"Do 50 jumping jacks!",
	"Tell a joke — if no one laughs, do the dare again!",
	"Record yourself doing a TikTok dance!",
}

var truths = []string{
	"What's the most embarrassing thing that's happened to you?",
	"What's a secret you've never told anyone in this group?",
	"What's the biggest lie you've ever told?",
	"Who's your celebrity crush?",
	"What's the weirdest dream you've ever had?",
	"What's your most irrational fear?",
	"What's the last thing you searched on Google?",
	"Have you ever cheated on a test?",
	"What's the pettiest thing you've ever done?",
	"What's a habit you have that you're embarrassed about?",
	"What's the most trouble you've ever gotten into?",
	"What's your worst habit?",
	"What's something you've never told your parents?",
	"What's the most childish thing you still do?",
	"Have you ever pretended to be sick to skip something?",
}

var insults = []string{
	"You're as useful as a white crayon.",
	"I'd call you sharp, but that would be offensive to pencils.",
	"You're like a cloud. When you disappear, it's a beautiful day.",
	"You bring everyone so much joy... when you leave the room.",
	"If laziness was an Olympic sport, you'd come in fourth — so you wouldn't have to walk up to the podium.",
	"You're not stupid — you just have bad luck thinking.",
	"I'm not saying you're slow, but you'd lose a race with a sloth.",
}

var flirtLines = []string{
	"Are you a magician? Because whenever I look at you, everyone else disappears. ✨",
	"Do you have a map? I keep getting lost in your eyes. 🗺️",
	"Is your name Wi-Fi? Because I'm really feeling a connection. 📶",
	"If being cute was a crime, you'd be in serious trouble. 🚨",
	"You must be tired, because you've been running through my mind all day. 💭",
	"Are you a parking ticket? Because you've got fine written all over you. 😉",
	"Do you believe in love at first sight, or should I walk by again? 👀",
}

func randomFrom(list []string) string {
	return list[rand.Intn(len(list))]
}

// mentionedOrSender returns the first @mention if present, else the sender.
func mentionedOrSender(v *events.Message) string {
	if v.Message != nil && v.Message.ExtendedTextMessage != nil {
		ctx := v.Message.ExtendedTextMessage.ContextInfo
		if ctx != nil && len(ctx.MentionedJID) > 0 {
			return ctx.MentionedJID[0]
		}
		if ctx != nil && ctx.Participant != nil {
			return *ctx.Participant
		}
	}
	return v.Info.Sender.String()
}

func sendWithMention(v *events.Message, text string, mention string) error {
	wac := client.Get()
	_, err := wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text:        &text,
			ContextInfo: &waProto.ContextInfo{MentionedJID: []string{mention}},
		},
	})
	return err
}

func complimentHandler(v *events.Message, args []string) error {
	target := mentionedOrSender(v)
	return sendWithMention(v, randomFrom(compliments), target)
}

func dareHandler(v *events.Message, args []string) error {
	return ReplyErr(v, "🎯 *Dare:* "+randomFrom(dares))
}

func truthHandler(v *events.Message, args []string) error {
	return ReplyErr(v, "❓ *Truth:* "+randomFrom(truths))
}

func insultHandler(v *events.Message, args []string) error {
	target := mentionedOrSender(v)
	return sendWithMention(v, randomFrom(insults), target)
}

func flirtHandler(v *events.Message, args []string) error {
	target := mentionedOrSender(v)
	if target == v.Info.Sender.String() {
		// no one mentioned — just send the line plainly
		return ReplyErr(v, "💌 "+randomFrom(flirtLines))
	}
	return sendWithMention(v, "💌 "+randomFrom(flirtLines), target)
}

// .ship — two mentions, one mention + sender, reply + sender, or two random group members
func shipHandler(v *events.Message, args []string) error {
	var a, b string

	if v.Message != nil && v.Message.ExtendedTextMessage != nil {
		ctx := v.Message.ExtendedTextMessage.ContextInfo
		if ctx != nil && len(ctx.MentionedJID) >= 2 {
			a, b = ctx.MentionedJID[0], ctx.MentionedJID[1]
		} else if ctx != nil && len(ctx.MentionedJID) == 1 {
			a, b = ctx.MentionedJID[0], v.Info.Sender.String()
		} else if ctx != nil && ctx.Participant != nil {
			a, b = *ctx.Participant, v.Info.Sender.String()
		}
	}

	if a == "" || b == "" {
		if !isGroup(v) {
			return ReplyErr(v, "❌ Mention two users, or reply to someone, to ship them.")
		}
		wac := client.Get()
		info, err := wac.GetGroupInfo(context.Background(), v.Info.Chat)
		if err != nil || len(info.Participants) < 2 {
			return ReplyErr(v, "❌ Not enough members to ship!")
		}
		idxs := rand.Perm(len(info.Participants))
		a = info.Participants[idxs[0]].JID.String()
		b = info.Participants[idxs[1]].JID.String()
	}

	seed := 0
	for _, c := range a + b {
		seed += int(c)
	}
	love := (seed * 7) % 101
	if love < 0 {
		love = -love
	}

	verdict := "Mostly chaos 😂"
	if love > 75 {
		verdict = "A strong match ❤️"
	} else if love > 40 {
		verdict = "Could work 🤝"
	}

	nameA := "@" + strings.Split(a, "@")[0]
	nameB := "@" + strings.Split(b, "@")[0]
	text := fmt.Sprintf("%s + %s = %d%%\n%s", nameA, nameB, love, verdict)

	wac := client.Get()
	_, err := wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text:        &text,
			ContextInfo: &waProto.ContextInfo{MentionedJID: []string{a, b}},
		},
	})
	return err
}

// ── BOMB GAME (solo — see note on tictactoe in chat) ────────────────────
//
// Note: this bot only responds to its owner/sudo users (see isOwner/IsSudo
// in module.go), so a true multiplayer game like Knight Bot's tictactoe
// (where any two group members take turns) isn't a good fit here — nobody
// else's plain-text moves would even reach the handler. Bomb works fine
// solo (open boxes, avoid the bomb), so that one's included; tictactoe was
// skipped for that reason.

type bombBox struct {
	isBomb bool
	opened bool
}

type bombGame struct {
	boxes     []bombBox
	startedAt time.Time
}

var (
	bombMu    sync.Mutex
	bombGames = make(map[string]*bombGame) // key: chat JID string
)

func renderBombBoard(g *bombGame, reveal bool) string {
	var b strings.Builder
	numberEmoji := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣"}
	for i, box := range g.boxes {
		switch {
		case box.opened && box.isBomb:
			b.WriteString("💥")
		case box.opened:
			b.WriteString("✅")
		case reveal && box.isBomb:
			b.WriteString("💣")
		default:
			b.WriteString(numberEmoji[i])
		}
		if (i+1)%3 == 0 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func bombHandler(v *events.Message, args []string) error {
	key := v.Info.Chat.String()

	bombMu.Lock()
	if _, exists := bombGames[key]; exists {
		bombMu.Unlock()
		return ReplyErr(v, "❌ A bomb game is already running here. Send a number 1-9, or *suren* to surrender.")
	}
	boxes := make([]bombBox, 9)
	boxes[rand.Intn(9)].isBomb = true
	g := &bombGame{boxes: boxes, startedAt: time.Now()}
	bombGames[key] = g
	bombMu.Unlock()

	text := "乂 *B O M B*\n\nSend a number *1-9* to open a box. Avoid the bomb!\nType *suren* to surrender.\n\n" +
		renderBombBoard(g, false)
	return ReplyErr(v, text)
}

// HandleBombMove — call this on every plain-text message (from
// main.go), it only acts if the sender has an active bomb game running
// in that chat.
func HandleBombMove(v *events.Message) {
	key := v.Info.Chat.String()

	bombMu.Lock()
	g, exists := bombGames[key]
	bombMu.Unlock()
	if !exists {
		return
	}
	if !isOwner(v) {
		return
	}

	text := strings.ToLower(strings.TrimSpace(extractText(v)))
	if text == "" {
		return
	}

	if text == "suren" || text == "surrender" {
		bombMu.Lock()
		delete(bombGames, key)
		bombMu.Unlock()
		_ = ReplyErr(v, "🏳️ *Surrendered!*\n\n"+renderBombBoard(g, true))
		return
	}

	n, err := strconv.Atoi(text)
	if err != nil || n < 1 || n > 9 {
		return
	}

	idx := n - 1
	bombMu.Lock()
	if g.boxes[idx].opened {
		bombMu.Unlock()
		return
	}
	g.boxes[idx].opened = true
	isBomb := g.boxes[idx].isBomb

	won := true
	for _, box := range g.boxes {
		if !box.isBomb && !box.opened {
			won = false
			break
		}
	}
	if isBomb || won {
		delete(bombGames, key)
	}
	board := renderBombBoard(g, isBomb)
	bombMu.Unlock()

	switch {
	case isBomb:
		_ = ReplyErr(v, "💥 *BOOM! Game over!*\n\n"+board)
	case won:
		_ = ReplyErr(v, "🎉 *You win! All safe boxes opened!*\n\n"+board)
	default:
		_ = ReplyErr(v, board)
	}
}

func init() {
	Register(ModuleInfo{
		Name:        "FunExtra",
		Description: "compliment, dare, truth, insult, flirt, ship, bomb",
		Commands: []CommandInfo{
			{Pattern: "compliment", Handler: complimentHandler, Sudo: true},
			{Pattern: "dare", Handler: dareHandler, Sudo: true},
			{Pattern: "truth", Handler: truthHandler, Sudo: true},
			{Pattern: "insult", Handler: insultHandler, Sudo: true},
			{Pattern: "flirt", Handler: flirtHandler, Sudo: true},
			{Pattern: "ship", Handler: shipHandler, Sudo: true},
			{Pattern: "bomb", Handler: bombHandler, Sudo: true},
		},
	})
}
