# 🤖 PBX Elite

> **WhatsApp Userbot** — Go port of PBX Main (Pyrogram) + SC Phoenix + PBX UC2  
> Built with [whatsmeow](https://go.mau.fi/whatsmeow)

---

## 📦 Setup

```bash
# 1. Clone
git clone <your-repo>
cd pbx_elite

# 2. Install deps
go mod tidy

# 3. Run
go run main.go
```

Scan the QR code or enter your phone number to pair.

---

## ⚙️ Config

Edit `config/config.go`:

```go
Prefix  = "."          // Command prefix
OwnerJID = "91XXXXXXXXXX@s.whatsapp.net"
```

---

## 📋 Commands

See **[COMMANDS.md](COMMANDS.md)** for the full list of all **381 commands** across **130 modules**.

---

## 🗂️ Modules Overview

| Category | Commands |
|----------|----------|
| Raid (direct) | `.raid` `.pbiraid` `.hiraid` `.imoji` `.oneword` `.shraid` + 12 more |
| Reply Raid | `.replyraid` `.preplyraid` `.ereplyraid` `.hreplyraid` `.shreplyraid` + stops |
| Custom Raid | `.addraid` `.myraid` `.customraid` `.craid` + manage |
| Group Tools | `.kick` `.ban` `.promote` `.demote` `.tagall` `.gmute` `.link` |
| Anti-Spam | `.antispam` `.antilink` `.antiflood` `.antibot` `.antidelete` |
| Media | `.sticker` `.toimage` `.save` `.ytdl` `.ytdlmp3` `.instadl` |
| Fun | `.joke` `.meme` `.quote` `.dice` `.hangman` `.tictactoe` |
| Utility | `.translate` `.tts` `.ocr` `.weather` `.wiki` `.ud` `.calc` |
| Auto Tools | `.autoreply` `.autopilot` `.autosave` `.autoread` `.autoreact` |
| Stickers | `.sticker` `.imgs` `.stktoimg` `.pbxanim` |
| Status/Bio | `.bio` `.status` `.profilepic` `.cloneprofile` |
| Notes | `.note` `.addnote` `.notes` `.delnote` |
| Broadcast | `.broadcast` `.gcast` |
| Security | `.pmsecurity` `.sudo` `.adminonly` `.chatlock` |
| AFK | `.afk` `.unafk` |
| + 115 more modules | See COMMANDS.md |

---

## 🏗️ Architecture

```
pbx_elite/
├── main.go              — Entry point, whatsmeow event loop
├── pairing.go           — QR + phone number pairing
├── config/              — Bot configuration
├── client/              — whatsmeow client singleton (client.Get())
├── database/            — Persistent storage layer
├── mongostore/          — MongoDB session store
├── session/             — Session key management
├── web/                 — Web panel (pairing, status)
└── modules/             — 140 Go files, 381 commands
    ├── module.go        — HandleMessage() dispatcher
    ├── utils.go         — Shared helpers (Reply, ReplyErr, strPtr…)
    ├── raidpools.go     — 3000+ raid strings (all pools)
    ├── raid.go          — Direct raid commands
    ├── replyraid.go     — Auto reply-raid system
    └── customraid.go    — Personal custom raid list
```

### How modules work

```go
// Every module registers itself in init()
func init() {
    Register(ModuleInfo{
        Name:        "MyModule",
        Description: "Does something",
        Commands: []CommandInfo{
            {Pattern: "cmd", Handler: myHandler, Sudo: true},
        },
    })
}

func myHandler(v *events.Message, args []string) error {
    return Reply(v, "Hello!")
}
```

`HandleMessage()` in `module.go` dispatches every incoming WhatsApp message to the matching handler.

---

## 🔒 Sudo System

- **Owner** — detected via `IsFromMe` flag (most reliable)
- **Sudo users** — managed with `.addsudo` / `.rmsudo`
- Commands with `Sudo: true` are restricted to owner + sudo list
- Commands with `Sudo: false` are open to all

---

## 📊 Stats

| Metric | Value |
|--------|-------|
| Total commands | **381** |
| Total modules | **130** |
| Go files | **140** |
| Raid string pools | **20 pools, 3000+ strings** |
| Pattern duplicates | **0** |
| Func name conflicts | **0** |

---

## 📝 Credits

Ported from:
- **PBX Main** — Original Python/Pyrogram userbot
- **SC Phoenix** — Extended command set  
- **PBX UC2** — Additional utilities

Built with ❤️ using [whatsmeow](https://go.mau.fi/whatsmeow) by tulir
