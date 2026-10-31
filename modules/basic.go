package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"PbxGb/client"
	"PbxGb/config"
	"PbxGb/database"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

const boxEnd = "└───────────────────⭓"

func pingText(latencyMs int64) string {
	p := config.C.Prefix
	return strings.Join([]string{
		"┌⭓⭓ " + BotNameFancy + " ⭓⭓┐",
		"│",
		"│ 🏓 *" + smallCaps("pong!") + "*",
		"│ ⚡ *" + smallCaps("bot:") + "* " + BotNameFancy,
		fmt.Sprintf("│ ⏱ *%s* %dms", smallCaps("latency:"), latencyMs),
		"│ ⏲ *" + smallCaps("uptime:") + "* " + FormatUptime(time.Since(StartTime)),
		"│ 📦 *" + smallCaps("version:") + "* v" + BotVersion,
		"│ ✅ *" + smallCaps("status:") + "* " + smallCaps("online & responsive"),
		"│",
		"│ > _" + smallCaps("type "+p+"menu for all commands") + "_ 🤖",
		"│",
		"└◦ " + smallCaps("powered by pbx") + " ◦ ──┘",
	}, "\n")
}

// .ping — measures the real send round-trip, then edits the message in place.
func pingHandler(v *events.Message, args []string) error {
	wac := client.Get()
	ctx := context.Background()

	probe := "🏓 ..."
	start := time.Now()
	resp, err := wac.SendMessage(ctx, v.Info.Chat, &waProto.Message{Conversation: &probe})
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return err
	}

	text := pingText(latency)
	edit := wac.BuildEdit(v.Info.Chat, resp.ID, &waProto.Message{Conversation: &text})
	if _, err := wac.SendMessage(ctx, v.Info.Chat, edit); err != nil {
		return ReplyErr(v, text)
	}
	return nil
}

// .alive
func aliveHandler(v *events.Message, args []string) error {
	p := config.C.Prefix
	afk := "ᴏꜰꜰ"
	if database.GetAFKState().Enabled {
		afk = "ᴏɴ 😴"
	}
	msg := strings.Join([]string{
		"┌───⭓ " + BotNameFancy + " ⭓───",
		"│ ✅ *" + smallCaps("i'm alive and running") + "*",
		"│",
		"│ 🤖 *" + smallCaps("bot:") + "* " + BotNameFancy,
		"│ 👤 *" + smallCaps("owner:") + "* +" + ownerNum(),
		"│ ⚙️ *" + smallCaps("prefix:") + "* " + p,
		"│ ⏲ *" + smallCaps("uptime:") + "* " + FormatUptime(time.Since(StartTime)),
		fmt.Sprintf("│ 📚 *%s* %d", smallCaps("commands:"), CommandCount()),
		"│ 🌙 *" + smallCaps("afk:") + "* " + afk,
		"│ 📦 *" + smallCaps("version:") + "* v" + BotVersion,
		"│",
		"│ > _" + smallCaps("type "+p+"ping for latency, "+p+"menu for commands") + "_",
		"│",
		"│ 📢 *" + smallCaps("updates:") + "* " + UpdateURL,
		"│ 💬 *" + smallCaps("support:") + "* " + ChatURL,
		"│ 👑 *" + smallCaps("owner:") + "* " + OwnerURL,
		boxEnd,
	}, "\n")
	return ReplyErr(v, msg)
}

type menuSection struct {
	icon  string
	title string
	cmds  []string
}

func menuSections(p string) []menuSection {
	return []menuSection{
		{"⚡", "general", []string{
			p + "ping — latency & uptime", p + "alive — bot status", p + "menu / " + p + "help — this menu",
		}},
		{"🎬", "animation", []string{
			p + "anim <name> — edit-style animation",
			p + "animlist — all animations",
			p + "randomanim — random animation",
			p + "love, " + p + "crush, " + p + "rain, " + p + "dino, " + p + "hack, " + p + "snake ...",
			p + "shoot, " + p + "cobra, " + p + "helicopter, " + p + "gf, " + p + "run, " + p + "thumb ... — ASCII art",
		}},
		{"🌙", "afk", []string{
			p + "afk <reason> — AFK on (auto-off when you write)",
			p + "unafk — AFK off",
			p + "afkstatus — AFK status",
		}},
		{"🔐", "pm permit", []string{
			p + "pmsec on/off — PM permit toggle",
			p + "approve / " + p + "allow — approve this chat or @user",
			p + "disapprove / " + p + "disallow",
			p + "block / " + p + "unblock",
			p + "allowlist — approved list",
			p + "setpmmsg <text> — custom warn message",
			p + "pmwarn <n> — max warnings",
			p + "pmreset — clear all warns/blocks",
		}},
		{"🎵", "download", []string{
			p + "song <name/url> — YouTube MP3", p + "yt <name/url> — YouTube MP4",
			p + "ytmp3 <url>", p + "bass — reply audio → bass boost", p + "img <query>",
		}},
		{"🌐", "utility", []string{
			p + "calc 2+2", p + "remind 10m text", p + "tinyurl <url>",
			p + "translate hi text", p + "weather Delhi",
		}},
		{"😂", "fun", []string{
			p + "joke", p + "quote", p + "fact", p + "toss", p + "dice", p + "8ball", p + "rps",
			p + "truth / " + p + "dare", p + "ship", p + "shayari",
		}},
		{"🎥", "media", []string{
			p + "vv — view-once save", p + "autovv on/off", p + "sticker / " + p + "s",
			p + "autosave on/off", p + "save — reply to media",
		}},
		{"🛡️", "protection", []string{
			p + "antidelete on/off", p + "antiviewonce on/off", p + "antilink on/off",
			p + "autoreact on/off", p + "reactemoji <emoji/random>", p + "callreject on/off",
			p + "callmsg <text>",
		}},
		{"👥", "group", []string{
			p + "tagall <msg>", p + "hidetag <msg>", p + "kick @user", p + "add 91xxx",
			p + "promote / " + p + "demote @user", p + "mute / " + p + "unmute",
			p + "groupinfo", p + "warn / " + p + "resetwarn", p + "gban / " + p + "ungban",
		}},
		{"🧹", "clean", []string{p + "clean <n> — delete last n messages"}},
		{"📝", "text tools", []string{
			p + "reverse, " + p + "wordcount, " + p + "palindrome, " + p + "vowelcount <text>",
			p + "caesar <shift> <text>, " + p + "morse, " + p + "binary, " + p + "piglatin",
			p + "mock, " + p + "aesthetic, " + p + "leetspeak, " + p + "smallcaps, " + p + "strikethrough, " + p + "zalgo",
			p + "wordfreq <text>",
		}},
		{"🔢", "math tools", []string{
			p + "percent <x> <y>, " + p + "tempconv <v> <C|F|K>, " + p + "roman <n>",
			p + "prime <n>, " + p + "factors <n>, " + p + "fibonacci <n>, " + p + "factorial <n>",
			p + "gcd <a> <b>, " + p + "tip <bill> <%>, " + p + "discount <price> <%>",
			p + "baseconv <n> <from> <to>, " + p + "numtowords <n>",
		}},
		{"🎲", "pbx fun", []string{
			p + "coinflip, " + p + "roulette, " + p + "slots, " + p + "luckynumber, " + p + "randomcolor",
			p + "simp, " + p + "stupid, " + p + "spiritanimal, " + p + "zodiac <d> <m>",
			p + "owner — owner & support links, " + p + "uptime",
		}},
		{"👑", "sudo", []string{
			p + "addsudo 91xxx", p + "removesudo 91xxx", p + "sudolist",
		}},
	}
}

func menuText(v *events.Message) string {
	p := config.C.Prefix
	name := v.Info.PushName
	if name == "" {
		name = "User"
	}

	var sb strings.Builder
	sb.WriteString("┌───⭓ " + BotNameFancy + " ⭓───\n")
	sb.WriteString("│ 🧠 *" + smallCaps("userbot control core") + "*\n")
	sb.WriteString("│ 👋 *" + smallCaps("welcome:") + "* " + strings.ToUpper(name) + "\n")
	sb.WriteString("│ 👤 *" + smallCaps("owner:") + "* +" + ownerNum() + "\n")
	sb.WriteString("│ ⚙️ *" + smallCaps("prefix:") + "* " + p + "\n")
	sb.WriteString("│ ⏲ *" + smallCaps("uptime:") + "* " + FormatUptime(time.Since(StartTime)) + "\n")
	sb.WriteString(fmt.Sprintf("│ 📚 *%s* %d\n", smallCaps("commands:"), CommandCount()))
	sb.WriteString("│ 📦 *" + smallCaps("version:") + "* v" + BotVersion + "\n")
	sb.WriteString(boxEnd + "\n")

	for _, s := range menuSections(p) {
		sb.WriteString("\n┌───⭓ " + s.icon + " " + smallCaps(s.title) + " ⭓───\n")
		for _, c := range s.cmds {
			sb.WriteString("│ ➤ " + c + "\n")
		}
		sb.WriteString(boxEnd + "\n")
	}

	sb.WriteString("\n┌───────────────────⭓\n")
	sb.WriteString("│ 🟢 *" + smallCaps("status:") + "* " + smallCaps("online · secure") + "\n")
	sb.WriteString("│ ⚡ *" + smallCaps("powered by pbx") + "*\n")
	sb.WriteString("│ 📢 " + UpdateURL + "\n")
	sb.WriteString("│ 💬 " + ChatURL + "\n")
	sb.WriteString("│ 👑 " + OwnerURL + "\n")
	sb.WriteString(boxEnd)
	return sb.String()
}

func menuHandler(v *events.Message, args []string) error {
	return ReplyErr(v, menuText(v))
}

func startHandler(v *events.Message, args []string) error {
	p := config.C.Prefix
	return ReplyErr(v, strings.Join([]string{
		"┌───⭓ " + BotNameFancy + " ⭓───",
		"│ ✅ " + smallCaps("bot is online!"),
		"│ 📌 " + smallCaps("type "+p+"menu to see all commands."),
		boxEnd,
	}, "\n"))
}

func init() {
	Register(ModuleInfo{
		Name:        "Basic",
		Description: "ping, alive, start, menu, help",
		Commands: []CommandInfo{
			{Pattern: "ping", Handler: pingHandler, Sudo: true},
			{Pattern: "alive", Handler: aliveHandler, Sudo: true},
			{Pattern: "start", Handler: startHandler, Sudo: true},
			{Pattern: "help", Handler: menuHandler, Sudo: true},
			{Pattern: "menu", Handler: menuHandler, Sudo: true},
		},
	})
}

// MenuGroup is the exported form of one menu section (used by the web dashboard).
type MenuGroup struct {
	Icon  string
	Title string
	Cmds  []string
}

// MenuData returns every command group for the dashboard.
func MenuData() []MenuGroup {
	secs := menuSections(config.C.Prefix)
	out := make([]MenuGroup, 0, len(secs))
	for _, s := range secs {
		out = append(out, MenuGroup{Icon: s.icon, Title: s.title, Cmds: s.cmds})
	}
	return out
}
