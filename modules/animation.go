package modules

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"PbxGb/client"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

var animationList = map[string][]string{
	"love":    {"❤️ I", "❤️ I Love", "❤️ I Love You", "❤️ I Love You <3"},
	"bad":     {"❤️ ᴍʏ", "💝 ᴍʏ ᴄᴜᴛᴇ", "💞 ᴍʏ ᴄᴜᴛᴇ ᴏᴡɴᴇʀ", "[⎯꯭̽🇨🇦꯭꯭ ⃪В꯭α꯭∂ ꯭м꯭υ꯭η꯭∂꯭α_꯭آآ⎯꯭ ꯭̽🌸](https://t.me/ll_BAD_MUNDA_ll)"},
	"rain":    {"🌬", "☁️", "🌩", "🌨", "🌧", "🌦", "🌨🌩🌦🌥⛅🌤"},
	"shizu":   {"😘😘ʙᴀʙʏ sᴜɴᴏ ɴᴀ😘😘", "😍ᴍᴇʀᴀ ʙᴀᴄʜᴀ 😍", "😍ʙᴀʙʏ ɪ ʟᴏᴠᴇ ʟᴏᴠᴇ ʟᴏᴠᴇ ʏᴏᴜ🥰", "😘ʙᴀʙʏ ɪ ᴍɪss ʏᴏᴜ sᴏ ᴍᴜᴄʜ🙁", "🫣ᴛᴜ ᴍᴇʀɪ ᴀ ᴊᴀᴀɴ ᴍᴇᴛɪ 🤗", "💋ʙᴀʙʏ ɪ ᴋɪss ʏᴏᴜ 🥰", "🙈sʜɪᴢᴜ ᴊᴀᴀɴ ɪ ʟᴏᴠᴇ ʏᴏᴜ ɪ ᴍɪss ʏᴏᴜ ɪ ᴋɪss ʏᴏᴜ💫🥰❤️"},
	"hmm":     {"┈┈╱▔▔▔▔▔╲┈┈┈HM┈HM\n┈╱┈┈╱▔╲╲╲▏┈┈┈HMMM\n╱┈┈╱━╱▔▔▔▔▔╲━╮┈┈\n▏┈▕┃▕╱▔╲╱▔╲▕╮┃┈┈\n▏┈▕╰━▏▊▕▕▋▕▕━╯┈┈\n╲┈┈╲╱▔╭╮▔▔┳╲╲┈┈┈\n┈╲┈┈▏╭━━━━╯▕▕┈┈┈\n┈┈╲┈╲▂▂▂▂▂▂╱╱┈┈┈"},
	"ahh":     {"ahh", "aahh", "aahhh", "aahhhh", "aahhhhh", "aahhhhhh", "aahhhhhhh", "aaahhhhhhhh"},
	"bomb":    {"▪️▪️▪️▪️\n▪️▪️▪️▪️\n▪️▪️▪️▪️\n▪️▪️▪️▪️\n▪️▪️▪️▪️", "💣💣💣💣\n▪️▪️▪️▪️\n▪️▪️▪️▪️\n▪️▪️▪️▪️\n▪️▪️▪️▪️", "▪️▪️▪️▪️\n💣💣💣💣\n▪️▪️▪️▪️\n▪️▪️▪️▪️\n▪️▪️▪️▪️", "▪️▪️▪️▪️\n▪️▪️▪️▪️\n💣💣💣💣\n▪️▪️▪️▪️\n▪️▪️▪️▪️", "▪️▪️▪️▪️\n▪️▪️▪️▪️\n▪️▪️▪️▪️\n▪️▪️▪️▪️\n💥💥💥💥", "`RIP PLOXXX......`"},
	"hack":    {"Looking for WhatsApp databases...", "User online: True\nTelegram access: True", "Hacking... 0%\n[░░░░░░░░░░░░░░░░░░░░]\nETA: 0m, 20s", "Hacking... 25%\n[█████░░░░░░░░░░░░░░░]\nETA: 0m, 15s", "Hacking... 50%\n[██████████░░░░░░░░░░]\nETA: 0m, 10s", "Hacking... 75%\n[███████████████░░░░░]\nETA: 0m, 5s", "Hacking... 100%\n[████████████████████]\nScanning file...", "Hacking complete!\n✅ File uploaded to server.\n`./DOWNLOADS/msgstore.db.crypt12`"},
	"dino":    {"`DIN DINNN.....`", "`🏃                        🦖`", "`🏃                     🦖`", "`🏃                  🦖`", "`🏃               🦖`", "`🏃            🦖`", "`🏃WORGH!   🦖`", "`🏃            🦖`", "`🏃               🦖`", "`🏃                  🦖`", "`🏃 Huh-Huh           🦖`", "`🏃         🦖`", "`HE'S GETTING CLOSER!!!`", "`🏃    🦖`", "`🏃  🦖`", "`Just give up`", "`🧎🦖`", "`-ENDED-`"},
	"theart":  {"❤️", "🧡", "💛", "💚", "💙", "💜", "🖤", "💘", "💝", "❤️", "🧡", "💛", "💝", "💜"},
	"wtf":     {"What", "What The", "What The F", "What The F Brah", "*What The F Brah* 😂"},
	"gangstar": {"EVERyBOdy", "iZ", "GangSTur", "UNtIL", "I", "ArRivE", "🔥🔥🔥", "EVERyBOdy iZ GangSTur UNtIL I ArRivE 🔥🔥🔥"},
	"charging": {"`Tesla Wireless Charging...\nDevice: Nokia 1100\nBattery: 10%`", "`Tesla Wireless Charging...\nDevice: Nokia 1100\nBattery: 30%`", "`Tesla Wireless Charging...\nDevice: Nokia 1100\nBattery: 50%`", "`Tesla Wireless Charging...\nDevice: Nokia 1100\nBattery: 70%`", "`Tesla Wireless Charging...\nDevice: Nokia 1100\nBattery: 90%`", "`Tesla Wireless Charging...\nDevice: Nokia 1100\nBattery: 100% ✅`"},
	"plain":   {"Wait for plane...", "✈-------------", "---✈----------", "------✈-------", "---------✈----", "------------✈-", "------------✈ 🛬"},
	"music":   {"starting player...", "⬤⬤⬤ 81%\n\n🎵 *Now Playing: Shape of You*\n\n`00:00` ▱▱▱▱▱▱▱▱▱▱ `00:10`\n\n`🔂` `⏮️` `⏪️` `▶️` `⏩️` `⏭️`", "⬤⬤⬤ 81%\n\n🎵 *Now Playing: Shape of You*\n\n`00:04` ▰▰▰▰▱▱▱▱▱▱ `00:10`\n\n`🔂` `⏮️` `⏪️` `⏸️` `⏩️` `⏭️`", "⬤⬤◯ 80%\n\n🎵 *Now Playing: Shape of You*\n\n`00:08` ▰▰▰▰▰▰▰▰▱▱ `00:10`\n\n`🔂` `⏮️` `⏪️` `⏸️` `⏩️` `⏭️`", "⬤⬤◯ 80%\n\n🎵 *Now Playing: Shape of You*\n\n`00:10` ▰▰▰▰▰▰▰▰▰▰ `00:10`\n\n`🔂` `⏮️` `⏪️` `⏺️` `⏩️` `⏭️`"},
	"snake":   {"◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️", "◻️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️", "◻️◻️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️", "◻️◻️◻️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️", "◻️◻️◻️◻️◻️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️", "◻️◻️◻️◻️◻️\n◼️◼️◼️◼️◻️\n◼️◼️◼️◼️◻️\n◼️◼️◼️◼️◻️\n◼️◼️◼️◼️◻️", "◻️◻️◻️◻️◻️\n◻️◼️◼️◼️◻️\n◻️◼️◼️◼️◻️\n◻️◼️◼️◼️◻️\n◻️◻️◻️◻️◻️", "◻️◻️◻️◻️◻️\n◻️◻️◼️◻️◻️\n◻️◻️◻️◻️◻️\n◻️◻️◻️◻️◻️\n◻️◻️◻️◻️◻️", "◻️◻️◻️◻️◻️\n◻️◼️◻️◼️◻️\n◻️◻️◻️◻️◻️\n◻️◼️◼️◼️◻️\n◻️◻️◻️◻️◻️"},
	"gm":      {"ɢᴏᴏᴅ ᴍᴏʀɴɪɴɢ...\n｡♥｡･ﾟ♡ﾟ･｡♥｡\n╱╱╱╱╱╱╱╭╮╱╱╱╭╮╱╭╮╭╮\n╭━┳━┳━┳╯┃╭━┳╋╋━┫╰┫╰╮\n┃╋┃╋┃╋┃╋┃┃┃┃┃┃╋┃┃┃╭┫\n╰━╯╱╱╱╱╱╱╱╱╱╱╰━╯\n｡♥｡･ﾟ♡ﾟ･｡♥｡"},
	"gn":      {"ɢᴏᴏᴅ ɴɪɢʜᴛ...\n｡♥｡･ﾟ♡ﾟ･｡♥｡\n╱╱╱╱╱╱╱╭╮╱╱╱╱╱╱╱╱╱╱╭╮\n╭━┳━┳━┳╯┃╭━━┳━┳┳┳━┳╋╋━┳┳━╮\n┃╋┃╋┃╋┃╋┃┃┃┃┃╋┃╭┫┃┃┃┃┃┃╋┃\n╰━╯╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╰━╯\n｡♥｡･ﾟ♡ﾟ･｡♥｡"},
	"nikal":   {"`⠀⠀⠀⣠⣶⡾⠏⠉⠙⠳⢦⡀⠀⠀⠀⢠⠞⠉⠙⠲⡀⠀`\nNikal", "`⠀⠀⠀⣠⣶⡾⠏⠉⠙⠳⢦⡀⠀⠀⠀⢠⠞⠉⠙⠲⡀⠀`\nLavde", "`⠀⠀⠀⣠⣶⡾⠏⠉⠙⠳⢦⡀⠀⠀⠀⢠⠞⠉⠙⠲⡀⠀`\nPehli Fursat Meeee", "`⠀⠀⠀⣠⣶⡾⠏⠉⠙⠳⢦⡀⠀⠀⠀⢠⠞⠉⠙⠲⡀⠀`\nlodu Nikal 😂"},
	"loveu":   {"╲╲╲┏━━┓╭━━━╮╱╱╱\n╲╲╲┗┓┏┛┃╭━╮┃╱╱╱\n╲╲╲╲┃┃┏┫┃╭┻┻┓╱╱\n╱╱╱┏╯╰╯┃╰┫┏━╯╱╱\n╱╱┏┻━┳┳┻━┫┗┓╱╱╱\n╱╱╰━┓┃┃╲┏┫┏┛╲╲╲\n╱╱╱╱┃╰╯╲┃┃┗━╮╲╲\n╱╱╱╱╰━━━╯╰━━┛╲╲", "░I░L░O░V░E░Y░O░U░", "╔ღ═╗╔╗\n╚╗╔╝║║ღ═╦╦╦═ ღ\n╔╝╚╗ ღ╚╣║║║║╠╣\n╚═ ღ╝╚═╩═╩ ღ╩═╝"},
	"ror":     {"Rooor", "Rooor Aahh", "Rooor Aahh Aahh", "Rooor Aahh Aahh Aahh 😂"},
	"tank":    {"█۞███████]▄▄▄▄▄▄▄▄▄▄▃\n▂▄▅█████████▅▄▃▂...\n[███████████████████]\n◥⊙▲⊙▲⊙▲⊙▲⊙▲⊙▲⊙◤"},
	"cat":     {"┈┈┏━╮╭━┓┈╭━━━━╮\n┈┈┃┏┗┛┓┃╭┫cat ┃\n┈┈╰┓▋▋┏╯╯╰━━━━╯\n┈╭━┻╮╲┗━━━━╮╭╮┈\n┈┃▎▎┃╲╲╲╲╲╲┣━╯┈\n┈╰━┳┻▅╯╲╲╲╲┃┈┈┈\n┈┈┈╰━┳┓┏┳┓┏╯┈┈┈\n┈┈┈┈┈┗┻┛┗┻┛┈┈┈┈"},
	"scam":    {"`Activates Witchcraft Commands Online....`", "`Search for This Person's Name...`", "`Online Witchcraft Performed Immediately`", "0%   ▎", "30%   ███▌", "60%   ███████", "90%   ████████████", "100%   █████████████", "*Target Successfully Scammed Online 🥴*"},
	"addsudo": {"Deploying........", "*Heroku Connecting To Latest Github Build*", "*Build started by user PBX Elite*", "*Restarting Heroku Server...*", "*State changed from up to starting*", "*Starting process with command ./start.sh*", "*State changed from starting to up*", "__INFO:PBXElite:Logged in successfully__", "__INFO:PBXElite:Successfully loaded all plugins__", "*Build Succeeded ✅*"},
}

// Anti-ban limits (same budget PBX Elite uses): WhatsApp can flag an account
// that edits one message too many times too fast.
const (
	animMaxFrames   = 8
	animMinInterval = 700 * time.Millisecond
	animCooldown    = 4 * time.Second
)

var (
	animMu   sync.Mutex
	animLast = map[string]time.Time{}
)

// capFrames evenly down-samples frames to at most max entries (keeps first/last).
func capFrames(frames []string, max int) []string {
	if len(frames) <= max || max < 2 {
		return frames
	}
	out := make([]string, 0, max)
	step := float64(len(frames)-1) / float64(max-1)
	for i := 0; i < max; i++ {
		out = append(out, frames[int(math.Round(float64(i)*step))])
	}
	return out
}

func animKeys() []string {
	keys := make([]string, 0, len(animationList))
	for k := range animationList {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// playFrames sends the first frame, then edits the SAME message in place
// (native WhatsApp edit via BuildEdit) through the remaining frames.
func playFrames(v *events.Message, frames []string) error {
	if len(frames) == 0 {
		return nil
	}

	chat := v.Info.Chat.String()
	animMu.Lock()
	if last, ok := animLast[chat]; ok {
		if wait := animCooldown - time.Since(last); wait > 0 {
			animMu.Unlock()
			return ReplyErr(v, fmt.Sprintf("⏳ Wait %ds before another animation.", int(wait.Seconds())+1))
		}
	}
	animLast[chat] = time.Now()
	animMu.Unlock()

	frames = capFrames(frames, animMaxFrames)
	wac := client.Get()

	first := frames[0]
	resp, err := wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		Conversation: &first,
	})
	if err != nil {
		return err
	}

	for _, frame := range frames[1:] {
		time.Sleep(animMinInterval)
		f := frame
		edit := wac.BuildEdit(v.Info.Chat, resp.ID, &waProto.Message{Conversation: &f})
		if _, err := wac.SendMessage(context.Background(), v.Info.Chat, edit); err != nil {
			// Edits aren't guaranteed — stop instead of spamming new messages.
			break
		}
	}
	return nil
}

// animEditHandler — .anim <name>
func animEditHandler(v *events.Message, args []string) error {
	if len(args) < 1 {
		return ReplyErr(v, fmt.Sprintf(
			"┌───⭓ 🎬 *ANIMATIONS* ⭓───\n│ %s\n└───────────────────⭓\n\n*Usage:* `.anim love`",
			strings.Join(animKeys(), ", "),
		))
	}

	frames, exists := animationList[strings.ToLower(args[0])]
	if !exists {
		return ReplyErr(v, "❌ Animation not found. Type `.animlist` to see the list.")
	}
	if len(frames) == 1 {
		return ReplyErr(v, frames[0])
	}
	return playFrames(v, frames)
}

// animListHandler — all animations
func animListHandler(v *events.Message, args []string) error {
	keys := animKeys()
	msg := fmt.Sprintf(
		"┌───⭓ 🎬 *ANIMATION LIST* ⭓───\n│ *Total:* %d\n│\n│ %s\n└───────────────────⭓\n\n*Usage:* `.anim <name>`\n*Example:* `.anim love`",
		len(keys),
		strings.Join(keys, " • "),
	)
	return ReplyErr(v, msg)
}

// randomAnimHandler
func randomAnimHandler(v *events.Message, args []string) error {
	keys := animKeys()
	return animEditHandler(v, []string{keys[rand.Intn(len(keys))]})
}

func boolPtr(b bool) *bool { return &b }

func init() {
	Register(ModuleInfo{
		Name:        "Animation",
		Description: "Full animation pack with message edit style",
		Commands: []CommandInfo{
			{Pattern: "anim", Handler: animEditHandler, Sudo: true},
			{Pattern: "animlist", Handler: animListHandler, Sudo: true},
			{Pattern: "randomanim", Handler: randomAnimHandler, Sudo: true},
		},
	})
}
