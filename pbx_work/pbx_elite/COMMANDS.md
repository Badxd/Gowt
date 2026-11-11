# PBX Elite — Command Reference

> WhatsApp userbot built with [whatsmeow](https://go.mau.fi/whatsmeow)
> **Go-based port of PBX Main (Pyrogram) + SC Phoenix + PBX UC2**

**145 modules · 450+ commands**

---

## Quick Start

```bash
git clone <repo>
cd pbx_elite
go mod tidy
go run main.go
```

Default prefix: `.`  (configurable in `config/config.go`)

🔒 = Owner/Sudo only  &nbsp; 🔓 = Everyone

---

## Modules

### Admin Controls
> Admin-only mode, group lock/unlock, add/remove/promote/demote members

| Command | Access |
|---------|--------|
| `.adminonly` | 🔒 |
| `.lockgroup` | 🔒 |
| `.unlockgroup` | 🔒 |
| `.removemember` | 🔒 |

### AFK
> AFK status: auto-reply when mentioned while away

| Command | Access |
|---------|--------|
| `.afk` | 🔓 |
| `.unafk` | 🔓 |

### Alive
> Bot health check and uptime status

| Command | Access |
|---------|--------|
| `.alive` | 🔒 |

### Animation
> Full animation pack with message edit style

| Command | Access |
|---------|--------|
| `.anim` | 🔒 |
| `.animlist` | 🔒 |
| `.randomanim` | 🔒 |

### Animoji
> Animated emoji and ASCII art reactions

| Command | Access |
|---------|--------|
| `.think` | 🔒 |
| `.lmao` | 🔒 |
| `.clock` | 🔒 |
| `.heartan` | 🔒 |
| `.muah` | 🔒 |
| `.fap` | 🔒 |
| `.ccry` | 🔒 |
| `.carry` | 🔒 |
| `.ded` | 🔒 |

### Antibot
> Auto-kick bot/spam accounts in groups

| Command | Access |
|---------|--------|
| `.antibot` | 🔒 |
| `.antibotlist` | 🔒 |

### AntiDelete
> Catch deleted messages

| Command | Access |
|---------|--------|
| `.antidelete` | 🔒 |

### AntiFlood
> Detect and act on message flooding in groups

| Command | Access |
|---------|--------|
| `.antiflood` | 🔒 |
| `.stopantiflood` | 🔒 |

### Anti-Link
> Auto-delete WhatsApp group invite links from non-admins

| Command | Access |
|---------|--------|
| `.antilink` | 🔒 |
| `.allowlink` | 🔒 |
| `.disallowlink` | 🔒 |

### Anti-Spam
> Detect and handle message spam in groups

| Command | Access |
|---------|--------|
| `.antispam` | 🔒 |

### AntiViewOnce
> Auto-save view-once media to Saved Messages

| Command | Access |
|---------|--------|
| `.antiviewonce` | 🔒 |

### Autopilot
> Auto-reply to trigger keywords when autopilot is enabled

| Command | Access |
|---------|--------|
| `.autopilot` | 🔒 |
| `.setreply` | 🔒 |
| `.delreply` | 🔒 |
| `.listreplies` | 🔒 |

### AutoReact
> Auto react to DM messages

| Command | Access |
|---------|--------|
| `.autoreact` | 🔒 |
| `.reactemoji` | 🔒 |

### Auto Read
> Automatically mark incoming messages as read

| Command | Access |
|---------|--------|
| `.autoread` | 🔒 |
| `.readall` | 🔒 |

### Auto Reply
> Set keyword-triggered automatic replies

| Command | Access |
|---------|--------|
| `.ar` | 🔒 |
| `.autoreply` | 🔒 |

### AutoSave
> Auto save media to self chat

| Command | Access |
|---------|--------|
| `.autosave` | 🔒 |

### Ban
> Kick and ban users from groups, with ban list enforcement

| Command | Access |
|---------|--------|
| `.ban` | 🔒 |
| `.unban` | 🔒 |
| `.banlist` | 🔒 |

### Base64 & Hash
> Base64 encode/decode and hash text

| Command | Access |
|---------|--------|
| `.b64en` | 🔒 |
| `.b64de` | 🔒 |
| `.hash` | 🔒 |

### Basic
> ping, alive, start, menu, help

| Command | Access |
|---------|--------|
| `.ping` | 🔒 |
| `.start` | 🔒 |
| `.menu` | 🔒 |
| `.helpme` | 🔓 |

### Bio
> Get or set the bot's WhatsApp about/bio

| Command | Access |
|---------|--------|
| `.getbio` | 🔒 |

### Blacklist
> Auto-delete messages containing blacklisted words

| Command | Access |
|---------|--------|
| `.addbl` | 🔒 |
| `.rmbl` | 🔒 |
| `.bllist` | 🔒 |

### Broadcast
> Send a message to all groups the bot is in

| Command | Access |
|---------|--------|
| `.bc` | 🔒 |
| `.broadcast` | 🔒 |
| `.bcwarn` | 🔒 |
| `.bcgroups` | 🔒 |
| `.forward` | 🔒 |

### Calculator
> Evaluate math expressions and perform math operations

| Command | Access |
|---------|--------|
| `.calculate` | 🔒 |
| `.math` | 🔒 |

### CallReject
> Auto reject calls + busy message to caller

| Command | Access |
|---------|--------|
| `.callreject` | 🔒 |
| `.callmsg` | 🔒 |

### Carbon
> Generate code screenshot (carbon.now.sh style)

| Command | Access |
|---------|--------|
| `.carbon` | 🔒 |

### Chatbot
> AI chatbot: .ai <question> | .aimode on/off

| Command | Access |
|---------|--------|
| `.ai` | 🔒 |
| `.aimode` | 🔒 |

### Chat Lock
> Lock/unlock a chat to ignore all commands in it

| Command | Access |
|---------|--------|
| `.chatlock` | 🔒 |
| `.chatunlock` | 🔒 |
| `.lockstatus` | 🔒 |

### Chat Tools
> Group description, name, photo, and info commands

| Command | Access |
|---------|--------|
| `.getname` | 🔒 |
| `.setpic` | 🔒 |
| `.chatdesc` | 🔓 |
| `.setchatdesc` | 🔒 |

### Clean
> Delete bot messages from chat

| Command | Access |
|---------|--------|
| `.clean` | 🔒 |

### CloneProfile
> Clone profile picture from another user, or set your own PFP

| Command | Access |
|---------|--------|
| `.setpfp` | 🔒 |
| `.clone` | 🔒 |

### ColorPicker
> Get color info from hex, RGB, or name

| Command | Access |
|---------|--------|
| `.color` | 🔒 |

### Contacts
> Number lookup, WhatsApp check, JID tools

| Command | Access |
|---------|--------|
| `.number` | 🔒 |
| `.checkwa` | 🔒 |
| `.myid` | 🔒 |
| `.chatid` | 🔒 |

### CountryInfo
> Get detailed country information

| Command | Access |
|---------|--------|
| `.country` | 🔒 |

### Crypto
> Cryptocurrency price lookup via CoinGecko

| Command | Access |
|---------|--------|
| `.crypto` | 🔒 |
| `.price` | 🔒 |

### CustomRaid
> Personal custom raid message list — add, manage, and raid with your own strings

| Command | Access |
|---------|--------|
| `.addraid` | 🔒 |
| `.addr` | 🔒 |
| `.rmraid` | 🔒 |
| `.rmr` | 🔒 |
| `.listraid` | 🔒 |
| `.raidlist` | 🔒 |
| `.delraid` | 🔒 |
| `.clearraid` | 🔒 |
| `.myraid` | 🔒 |
| `.customraid` | 🔒 |
| `.craid` | 🔒 |

### Random
> Dice, coin flip, random number, pick, 8ball

| Command | Access |
|---------|--------|
| `.flip` | 🔒 |
| `.rng` | 🔒 |
| `.pick` | 🔒 |

### Download
> YouTube song/video download + bass boost + image search

| Command | Access |
|---------|--------|
| `.song` | 🔒 |
| `.ytmp3` | 🔒 |
| `.yt` | 🔒 |
| `.bass` | 🔒 |
| `.img` | 🔒 |

### Echo
> Echo (repeat) every message from a specific user

| Command | Access |
|---------|--------|
| `.echo` | 🔒 |
| `.rmecho` | 🔒 |
| `.listecho` | 🔒 |

### Emoji Tools
> Add, remove, or count emoji in text

| Command | Access |
|---------|--------|
| `.emojify` | 🔒 |
| `.demojify` | 🔒 |
| `.ecount` | 🔒 |

### Message Tools
> View quoted message content, message info, copy messages

| Command | Access |
|---------|--------|
| `.quoted` | 🔒 |
| `.msginfo` | 🔒 |
| `.copy` | 🔒 |

### Word Filter
> Auto-delete messages containing banned words in a group

| Command | Access |
|---------|--------|
| `.filter` | 🔒 |

### Fonts
> Convert text to fancy Unicode fonts

| Command | Access |
|---------|--------|
| `.font` | 🔒 |
| `.fontlist` | 🔒 |

### Forward
> Forward quoted messages to other chats/numbers

| Command | Access |
|---------|--------|
| `.fwd` | 🔒 |

### Fun
> Fun commands — joke, quote, fact, games

| Command | Access |
|---------|--------|
| `.joke` | 🔒 |
| `.fact` | 🔒 |
| `.toss` | 🔒 |
| `.dice` | 🔒 |
| `.8ball` | 🔒 |
| `.rps` | 🔒 |

### FunExtra
> Extra fun and abuse commands

| Command | Access |
|---------|--------|
| `.slap` | 🔒 |
| `.habuse` | 🔒 |
| `.randi` | 🔒 |
| `.noob` | 🔒 |
| `.chu` | 🔒 |
| `.fuk` | 🔒 |

### Broadcast
> Broadcast messages to all groups or users

| Command | Access |
|---------|--------|
| `.gcast` | 🔒 |
| `.gcastusers` | 🔒 |

### Get Profile Picture
> Fetch profile picture URL of a user

| Command | Access |
|---------|--------|
| `.getpp` | 🔒 |
| `.pfp` | 🔒 |

### Ghost Mode
> Hide read receipts, online status, and typing indicator

| Command | Access |
|---------|--------|
| `.ghost` | 🔒 |

### Glitch
> Apply Zalgo/glitch text effect

| Command | Access |
|---------|--------|
| `.glitch` | 🔒 |

### Greetings
> Group welcome/goodbye message management

| Command | Access |
|---------|--------|
| `.setwelcome` | 🔒 |
| `.setgoodbye` | 🔒 |
| `.delwelcome` | 🔒 |
| `.delgoodbye` | 🔒 |
| `.greetinfo` | 🔒 |

### Group Description
> Get or set group description

| Command | Access |
|---------|--------|
| `.setdesc` | 🔒 |
| `.getdesc` | 🔒 |

### Group Info
> Get group info, invite links, set group name, list members

| Command | Access |
|---------|--------|
| `.ginfo` | 🔒 |
| `.invitelink` | 🔒 |
| `.revokeinvite` | 🔒 |
| `.members` | 🔒 |

### Group Link
> Get or revoke/reset group invite link

| Command | Access |
|---------|--------|
| `.glink` | 🔒 |

### GroupStats
> Track and display group message statistics

| Command | Access |
|---------|--------|
| `.stats` | 🔒 |
| `.topmembers` | 🔒 |
| `.mystats` | 🔒 |
| `.resetstats` | 🔒 |

### GroupTools
> Group management — gmute, gban, tagall, kick, promote, demote

| Command | Access |
|---------|--------|
| `.kick` | 🔒 |
| `.add` | 🔒 |
| `.promote` | 🔒 |
| `.demote` | 🔒 |
| `.groupinfo` | 🔒 |
| `.gmute` | 🔒 |
| `.gunmute` | 🔒 |
| `.gmutechat` | 🔒 |
| `.gunmutechat` | 🔒 |
| `.gtag` | 🔒 |
| `.gban` | 🔒 |
| `.ungban` | 🔒 |
| `.gbanlist` | 🔒 |

### Hangman
> Play hangman in WhatsApp

| Command | Access |
|---------|--------|
| `.hm` | 🔒 |

### Help
> List all modules and commands

| Command | Access |
|---------|--------|
| `.help` | 🔒 |
| `.cmds` | 🔒 |

### ImageEdit
> Image effects: invert, greyscale, pixelate, noise

| Command | Access |
|---------|--------|
| `.invert` | 🔒 |
| `.grey` | 🔒 |
| `.pixelate` | 🔒 |
| `.noise` | 🔒 |

### Image to Sticker
> Convert images to WhatsApp stickers (512×512 WebP)

| Command | Access |
|---------|--------|
| `.imgsticker` | 🔒 |
| `.imgs` | 🔒 |

### Social Downloader
> Download media from Instagram, TikTok, Facebook, Twitter via yt-dlp

| Command | Access |
|---------|--------|
| `.igdl` | 🔒 |
| `.fbdl` | 🔒 |
| `.tikdl` | 🔒 |

### IPInfo
> Get geolocation and network info for an IP or domain

| Command | Access |
|---------|--------|
| `.ipinfo` | 🔒 |

### Leave/Join Group
> Leave current group, leave all groups, or join via invite link

| Command | Access |
|---------|--------|
| `.leave` | 🔒 |
| `.leaveall` | 🔒 |
| `.joingroup` | 🔒 |

### Location
> Send location pins and coordinates

| Command | Access |
|---------|--------|
| `.location` | 🔒 |
| `.loc` | 🔒 |
| `.coords` | 🔒 |
| `.liveloc` | 🔒 |

### Lyrics
> Fetch song lyrics

| Command | Access |
|---------|--------|
| `.lyrics` | 🔒 |

### MathTools
> Math utilities: factorial, fibonacci, prime, sqrt, power

| Command | Access |
|---------|--------|
| `.fib` | 🔒 |
| `.prime` | 🔒 |
| `.sqrt` | 🔒 |
| `.pow` | 🔒 |

### Media Filter
> Auto-delete specific media types (image/video/audio/etc.) in groups

| Command | Access |
|---------|--------|
| `.mediafilter` | 🔒 |
| `.mfilter` | 🔒 |

### Media Info
> Show metadata of a quoted media message

| Command | Access |
|---------|--------|
| `.mediainfo` | 🔒 |

### Meme
> Fetch random memes from Reddit

| Command | Access |
|---------|--------|
| `.meme` | 🔒 |
| `.dankmeme` | 🔒 |

### Mention All
> Hidden tag all members, kickme, leave group

| Command | Access |
|---------|--------|
| `.kickme` | 🔓 |

### Mentions
> Track and view when the bot is @mentioned across all chats

| Command | Access |
|---------|--------|
| `.mentions` | 🔒 |
| `.clearmention` | 🔒 |

### Morse
> Encode/decode Morse code

| Command | Access |
|---------|--------|
| `.morse` | 🔒 |
| `.unmorse` | 🔒 |

### Mute
> Soft-mute users in a group (auto-delete their messages)

| Command | Access |
|---------|--------|
| `.mute` | 🔒 |
| `.unmute` | 🔒 |
| `.mutelist` | 🔒 |

### User Stats
> Track and display per-user message statistics

| Command | Access |
|---------|--------|
| `.topusers` | 🔒 |

### News
> Fetch latest news headlines

| Command | Access |
|---------|--------|
| `.news` | 🔒 |

### Nicknames
> Assign and manage per-user nicknames

| Command | Access |
|---------|--------|
| `.setnick` | 🔒 |
| `.nick` | 🔒 |
| `.delnick` | 🔒 |
| `.nicklist` | 🔒 |

### Notes
> Save and retrieve per-chat notes

| Command | Access |
|---------|--------|
| `.addnote` | 🔒 |
| `.note` | 🔓 |
| `.notes` | 🔓 |
| `.delnote` | 🔒 |
| `.clearnotes` | 🔒 |

### OCR
> Extract text from images using OCR

| Command | Access |
|---------|--------|
| `.ocr` | 🔒 |

### Paste
> Paste text to dpaste.com and get a link

| Command | Access |
|---------|--------|
| `.paste` | 🔒 |

### Pin
> Pin or unpin messages in a group

| Command | Access |
|---------|--------|
| `.pin` | 🔒 |
| `.unpin` | 🔒 |

### PingInfo
> Bot uptime, sysinfo, public IP

| Command | Access |
|---------|--------|
| `.uptime` | 🔒 |
| `.sysinfo` | 🔒 |
| `.ip` | 🔒 |

### PMSecurity
> PM Permit — persistent DB-backed warnings, approvals, and auto-block

| Command | Access |
|---------|--------|
| `.pmsec` | 🔒 |
| `.pmstatus` | 🔒 |
| `.approve` | 🔒 |
| `.allow` | 🔒 |
| `.disapprove` | 🔒 |
| `.disallow` | 🔒 |
| `.block` | 🔒 |
| `.unblock` | 🔒 |
| `.allowlist` | 🔒 |
| `.approvelist` | 🔒 |
| `.setpmmsg` | 🔒 |
| `.pmwarn` | 🔒 |
| `.pmreset` | 🔒 |

### Poll
> Create WhatsApp polls in groups or DMs

| Command | Access |
|---------|--------|
| `.poll` | 🔒 |
| `.multipoll` | 🔒 |

### Presence
> Fake online/offline/typing/recording presence

| Command | Access |
|---------|--------|
| `.online` | 🔒 |
| `.offline` | 🔒 |
| `.typing` | 🔒 |
| `.recording` | 🔒 |
| `.lastseen` | 🔒 |

### Profile
> Profile management: name, bio, pfp

| Command | Access |
|---------|--------|
| `.setbio` | 🔒 |
| `.getpfp` | 🔒 |
| `.clonebio` | 🔒 |
| `.clonename` | 🔒 |
| `.setgname` | 🔒 |

### ProfilePic
> Set or delete the bot's profile picture (get is in getpp.go)

| Command | Access |
|---------|--------|
| `.setpp` | 🔒 |
| `.delpp` | 🔒 |

### Purge
> Delete messages: single delete or bulk purge of bot messages

| Command | Access |
|---------|--------|
| `.delete` | 🔒 |
| `.delmsg` | 🔒 |
| `.purge` | 🔒 |

### QRCode
> Generate and decode QR codes

| Command | Access |
|---------|--------|
| `.qr` | 🔒 |
| `.qrdecode` | 🔒 |

### Quote
> Fetch a random inspirational quote

| Command | Access |
|---------|--------|
| `.quote` | 🔒 |

### Raid
> Flood a user with messages from various pools (raid, pbiraid, hiraid, flirt, bday, shayri, emoji, oneword …)

| Command | Access |
|---------|--------|
| `.raid` | 🔒 |
| `.pbiraid` | 🔒 |
| `.hiraid` | 🔒 |
| `.imoji` | 🔒 |
| `.oneword` | 🔒 |
| `.shraid` | 🔒 |
| `.pbishraid` | 🔒 |
| `.engshraid` | 🔒 |
| `.hishraid` | 🔒 |
| `.flirt` | 🔒 |
| `.pflirt` | 🔒 |
| `.hflirt` | 🔒 |
| `.eflirt` | 🔒 |
| `.bday` | 🔒 |
| `.pbday` | 🔒 |
| `.hbday` | 🔒 |
| `.ebday` | 🔒 |
| `.multiraid` | 🔒 |

### React
> React to messages with emoji

| Command | Access |
|---------|--------|
| `.react` | 🔒 |
| `.reactall` | 🔒 |

### Reminder
> Set a timed reminder in any chat

| Command | Access |
|---------|--------|
| `.remind` | 🔒 |

### ReplyRaid
> Auto-reply on every message from a target user with raid strings

| Command | Access |
|---------|--------|
| `.replyraid` | 🔒 |
| `.dreplyraid` | 🔒 |
| `.preplyraid` | 🔒 |
| `.dpreplyraid` | 🔒 |
| `.ereplyraid` | 🔒 |
| `.dereplyraid` | 🔒 |
| `.hreplyraid` | 🔒 |
| `.dhreplyraid` | 🔒 |
| `.shreplyraid` | 🔒 |
| `.dshreplyraid` | 🔒 |
| `.listreplyraid` | 🔒 |

### TextTools
> Text manipulation: reverse, upper, lower, mock

| Command | Access |
|---------|--------|
| `.reverse` | 🔒 |
| `.upper` | 🔒 |
| `.lower` | 🔒 |
| `.mock` | 🔒 |

### Revoke
> Delete/revoke a message (reply to target message)

| Command | Access |
|---------|--------|
| `.revoke` | 🔒 |
| `.del` | 🔒 |

### Save File
> Save media messages to local disk

| Command | Access |
|---------|--------|
| `.save` | 🔒 |
| `.saved` | 🔒 |
| `.delsave` | 🔒 |

### Scheduler
> Schedule a message to be sent after a delay

| Command | Access |
|---------|--------|
| `.schedule` | 🔒 |
| `.schedulelist` | 🔒 |
| `.cancelschedule` | 🔒 |

### Screenshot
> Take a screenshot of any website

| Command | Access |
|---------|--------|
| `.ss` | 🔒 |

### Group Settings
> Lock/unlock group, manage group settings

| Command | Access |
|---------|--------|
| `.lock` | 🔒 |
| `.unlock` | 🔒 |
| `.gcopen` | 🔒 |
| `.gcclose` | 🔒 |
| `.joinapproval` | 🔒 |
| `.setname` | 🔒 |

### Shayari
> Random Shayari Sender

| Command | Access |
|---------|--------|
| `.shayari` | 🔒 |
| `.stopshayari` | 🔒 |

### Slow Mode
> Per-chat cooldown between messages from the same user

| Command | Access |
|---------|--------|
| `.slowmode` | 🔒 |

### Spam
> Spam and flood tools (bigspam, galispam, multispam)

| Command | Access |
|---------|--------|
| `.bigspam` | 🔒 |
| `.stopbigspam` | 🔒 |
| `.galispam` | 🔒 |
| `.stopgalispam` | 🔒 |
| `.multispam` | 🔒 |

### Speedtest
> Test server download speed

| Command | Access |
|---------|--------|
| `.speedtest` | 🔒 |

### Status
> Set or get the bot's WhatsApp about/bio text

| Command | Access |
|---------|--------|
| `.setstatus` | 🔒 |
| `.getstatus` | 🔒 |

### Status Tracker
> Track and view contact status updates

| Command | Access |
|---------|--------|
| `.statusview` | 🔒 |
| `.sv` | 🔒 |
| `.clearstatus` | 🔒 |

### Sticker Tools
> Steal sticker, convert sticker to image, show sticker info

| Command | Access |
|---------|--------|
| `.steal` | 🔒 |
| `.stktoimg` | 🔒 |
| `.stickername` | 🔒 |

### Sticker
> Convert image/video to sticker

| Command | Access |
|---------|--------|
| `.sticker` | 🔒 |
| `.s` | 🔒 |

### Sticker Tools
> Convert stickers to images, tag sticker packs

| Command | Access |
|---------|--------|
| `.toimage` | 🔒 |
| `.stickerpack` | 🔒 |
| `.getsticker` | 🔒 |

### Sudo
> Manage sudo (elevated) users who can run protected commands

| Command | Access |
|---------|--------|
| `.addsudo` | 🔒 |
| `.delsudo` | 🔒 |
| `.sudolist` | 🔒 |

### Tag
> Tag all members, admins, or silently notify all in a group

| Command | Access |
|---------|--------|
| `.tag` | 🔒 |
| `.hidetag` | 🔒 |
| `.tagadmins` | 🔒 |

### TagAll
> Mention all or admin members in a group

| Command | Access |
|---------|--------|
| `.tagall` | 🔒 |
| `.tagadmin` | 🔒 |
| `.listmembers` | 🔒 |

### Temp Ban
> Temporarily ban (kick + track) users from a group

| Command | Access |
|---------|--------|
| `.tempban` | 🔒 |
| `.tban` | 🔒 |
| `.tempbanlist` | 🔒 |

### Text Format Extra
> Extra text transforms: title, sponge, clap, small, fancy, mirror

| Command | Access |
|---------|--------|
| `.title` | 🔒 |
| `.sponge` | 🔒 |
| `.clap` | 🔒 |
| `.small` | 🔒 |
| `.fancy` | 🔒 |
| `.mirror` | 🔒 |

### URLTools
> Shorten and expand URLs

| Command | Access |
|---------|--------|
| `.shorten` | 🔒 |
| `.unshorten` | 🔒 |

### Translate
> Translate text using Google Translate

| Command | Access |
|---------|--------|
| `.tr` | 🔒 |

### FunGames
> Fun rating/game commands: ship, rate, truefake

| Command | Access |
|---------|--------|
| `.truefake` | 🔒 |
| `.rate` | 🔒 |
| `.ship` | 🔒 |
| `.gayrate` | 🔒 |
| `.pp` | 🔒 |

### TTS
> Text-to-speech via Google TTS

| Command | Access |
|---------|--------|
| `.tts` | 🔒 |

### UrbanDict
> Look up words on Urban Dictionary

| Command | Access |
|---------|--------|
| `.ud` | 🔒 |

### User Info
> Get detailed information about a WhatsApp user

| Command | Access |
|---------|--------|
| `.userinfo` | 🔒 |
| `.uinfo` | 🔒 |

### Utility
> calc, remind, tinyurl, translate, weather

| Command | Access |
|---------|--------|
| `.tinyurl` | 🔒 |

### vCard
> Send WhatsApp contact cards

| Command | Access |
|---------|--------|
| `.vcard` | 🔒 |
| `.vcardme` | 🔒 |
| `.vcardtag` | 🔒 |

### ViewOnce
> Save view-once media silently + auto save + 🫣 emoji save

| Command | Access |
|---------|--------|
| `.vv` | 🔒 |
| `.autovv` | 🔒 |

### Warnings
> Warning system with auto-kick after threshold

| Command | Access |
|---------|--------|
| `.warn` | 🔒 |
| `.warns` | 🔒 |
| `.resetwarn` | 🔒 |
| `.setwarnlimit` | 🔒 |
| `.warnlimit` | 🔒 |
| `.warnlist` | 🔒 |

### Weather
> Get current weather for a city

| Command | Access |
|---------|--------|
| `.weather` | 🔒 |

### Welcome/Goodbye
> Auto-send custom join/leave messages in groups

| Command | Access |
|---------|--------|
| `.welcome` | 🔒 |
| `.goodbye` | 🔒 |

### WhoIs
> User info lookup, bio management, display name

| Command | Access |
|---------|--------|
| `.whois` | 🔒 |
| `.bio` | 🔒 |
| `.name` | 🔒 |

### Wikipedia
> Search Wikipedia for information

| Command | Access |
|---------|--------|
| `.wiki` | 🔒 |

### Word Count
> Count words, chars, lines in text

| Command | Access |
|---------|--------|
| `.wc` | 🔒 |
| `.count` | 🔒 |

### YouTube
> YouTube search and audio download (requires yt-dlp)

| Command | Access |
|---------|--------|
| `.yts` | 🔒 |
| `.ytdl` | 🔒 |

### YTMedia
> Download YouTube audio (.mp3) and video (.mp4)

| Command | Access |
|---------|--------|
| `.mp3` | 🔒 |
| `.mp4` | 🔒 |

### YTInfo
> Get detailed YouTube video info (requires yt-dlp)

| Command | Access |
|---------|--------|
| `.ytinfo` | 🔒 |

---

## Raid Commands — Full Details

### Direct Raid
```
.raid <count> <phone>       — English abuse (1083 strings)
.pbiraid <count> <phone>    — Punjabi (14 strings)
.hiraid <count> <phone>     — Hindi (121 strings)
.imoji <count> <phone>      — Emoji (223 strings)
.oneword <count> <phone>    — One-word (178 strings)
.shraid <count> <phone>     — Shayari (39 strings)
.pbishraid <count> <phone>  — Punjabi Shayari (11 strings)
.engshraid <count> <phone>  — English Shayari (11 strings)
.hishraid <count> <phone>   — Hindi Shayari (11 strings)
.flirt <count> <phone>      — Flirt English (75 strings)
.pflirt <count> <phone>     — Flirt Punjabi (75 strings)
.hflirt <count> <phone>     — Flirt Hindi (75 strings)
.eflirt <count> <phone>     — Flirt Extra (75 strings)
.bday <count> <phone>       — Birthday English (30 strings)
.pbday <count> <phone>      — Birthday Punjabi (30 strings)
.hbday <count> <phone>      — Birthday Hindi (29 strings)
.ebday <count> <phone>      — Birthday Extra (30 strings)
.multiraid <count> <p1> <p2> ... — Raid multiple targets
```

### Reply Raid (auto-reply on every target message)
```
.replyraid <phone>     — Start English reply raid
.dreplyraid <phone>    — Stop English reply raid
.preplyraid <phone>    — Start Punjabi reply raid
.dpreplyraid <phone>   — Stop Punjabi reply raid
.ereplyraid <phone>    — Start English abuse reply raid
.dereplyraid <phone>   — Stop English abuse reply raid
.hreplyraid <phone>    — Start Hindi reply raid
.dhreplyraid <phone>   — Stop Hindi reply raid
.shreplyraid <phone>   — Start Shayari reply raid
.dshreplyraid <phone>  — Stop Shayari reply raid
.listreplyraid         — List all active reply raids
```

### Custom Raid (personal message list)
```
.addraid <text>  / .addr <text>     — Add to your list (max 50)
.rmraid <n>      / .rmr <n>          — Remove by index
.listraid        / .raidlist          — Show your list
.delraid         / .clearraid         — Clear entire list
.myraid <count> <phone>               — Raid with your list
.customraid / .craid <count> <phone>  — Aliases
```

---

## Group Extras

### GroupExtras
> Member counts, phone list, admin list, group management

| Command | Args | Access | Description |
|---------|------|--------|-------------|
| `.membercount` | — | 🔒 | Show member count in current group |
| `.vccount` | — | 🔒 | Show active voice-call count (tracked in memory) |
| `.phonelist` | — | 🔒 | List all member phone numbers in group |
| `.mygroups` | — | 🔒 | List all groups the bot is in |
| `.admins` | — | 🔒 | Show all admins of current group |
| `.groupage` | — | 🔒 | Show group creation date and age |
| `.gsettings` | — | 🔒 | Show group settings (announce, locked, ephemeral) |
| `.removemember` | `@user` | 🔒 | Remove a member from the group |
| `.addmember` | `number` | 🔒 | Add a number to the group |

---

## Call Tools

### CallTools
> Call history tracking, callback, voice call utilities

| Command | Args | Access | Description |
|---------|------|--------|-------------|
| `.callhistory` | — | 🔒 | Show last 10 incoming calls |
| `.callinfo` | — | 🔒 | Info about last received call |
| `.callback` | — | 🔒 | Initiate callback to last caller |
| `.rejectlast` | — | 🔒 | Reject last incoming call |
| `.vcping` | — | 🔒 | Show active VC participants (memory-tracked) |

> **Note:** Call tracking is automatic — bot tracks `CallOffer`/`CallTerminate` events in memory.

---

## Star / Unstar

### StarMsg
> Star and unstar messages

| Command | Args | Access | Description |
|---------|------|--------|-------------|
| `.star` | _(reply)_ | 🔒 | Star the quoted message |
| `.unstar` | _(reply)_ | 🔒 | Unstar the quoted message |
| `.starred` / `.starlist` | — | 🔒 | List starred messages (from local store) |

---

## Message Tools

### MsgTools
> Send messages, forward, get IDs, mark read, delete

| Command | Args | Access | Description |
|---------|------|--------|-------------|
| `.sendto` | `number text` | 🔒 | Send message directly to a phone number |
| `.fwdto` | `number` _(reply)_ | 🔒 | Forward quoted message to a number |
| `.getmsgid` | _(reply)_ | 🔒 | Get message ID of quoted message |
| `.quotedinfo` | _(reply)_ | 🔒 | Show full info of quoted message |
| `.markread` | — | 🔒 | Mark current chat as read |
| `.delforme` | _(reply)_ | 🔒 | Delete message for yourself |
| `.savecontact` | `@user` or `number` | 🔒 | Generate vCard for a number |
| `.sendtoall` | `text` | 🔒 | Broadcast text to all DM chats (capped 20, 1s delay) |

---

## Profile Tools

### ProfileTools
> Manage your profile and view others' profiles

| Command | Args | Access | Description |
|---------|------|--------|-------------|
| `.setbio` | `text` | 🔒 | Set your WhatsApp status/bio |
| `.setmypp` | _(reply image)_ | 🔒 | Set your profile photo |
| `.delmypp` | — | 🔒 | Remove your profile photo |
| `.getpp` | `@user` or reply | 🔒 | Get profile photo of a user |
| `.getname` | `@user` | 🔒 | Get display name of a user |
| `.getbio` | `@user` | 🔒 | Get status/bio of a user |
| `.copyprofile` | `@user` | 🔒 | Copy another user's name and photo |
| `.vcard` | `@user` | 🔒 | Generate vCard contact for a user |

---

## Status Tools

### StatusTools
> Post and manage WhatsApp Stories (status)

| Command | Args | Access | Description |
|---------|------|--------|-------------|
| `.setstatus` | `text` | 🔒 | Post a text WhatsApp Story |
| `.statusimg` | _(reply image)_ | 🔒 | Post image as WhatsApp Story |
| `.statusvid` | _(reply video)_ | 🔒 | Post video as WhatsApp Story |
| `.mystatus` / `.myprofile` | — | 🔒 | Show your current bio/status text |

---

## Chat Tools

### ChatTools
> Archive, pin, mute, disappear, and inspect chats

| Command | Args | Access | Description |
|---------|------|--------|-------------|
| `.archive` | — | 🔒 | Archive current chat |
| `.unarchive` | — | 🔒 | Unarchive current chat |
| `.pin` | — | 🔒 | Pin current chat |
| `.unpin` | — | 🔒 | Unpin current chat |
| `.mute` | `[minutes]` | 🔒 | Mute current chat (default 8h) |
| `.unmute` | — | 🔒 | Unmute current chat |
| `.disappear` / `.vanish` | `[seconds]` | 🔒 | Set disappearing messages timer |
| `.chatinfo` | — | 🔒 | Show chat metadata and settings |

---

## Group Desc & Links

### GroupDesc
> Group description, subject, links, join/leave, roles

| Command | Args | Access | Description |
|---------|------|--------|-------------|
| `.setdesc` | `text` | 🔒 | Set group description |
| `.getdesc` | — | 🔒 | Show current group description |
| `.setsubject` / `.rename` | `text` | 🔒 | Rename the group |
| `.grouplink` / `.invitelink` | — | 🔒 | Get group invite link |
| `.revokelink` | — | 🔒 | Revoke current invite link |
| `.groupjoin` / `.joingroup` | `link` | 🔒 | Join a group via invite link |
| `.groupleave` / `.leavegroup` | — | 🔒 | Leave the current group |
| `.setgrouppp` | _(reply image)_ | 🔒 | Set group profile photo |
| `.delgrouppp` | — | 🔒 | Remove group profile photo |
| `.groupreport` / `.greport` | — | 🔒 | Report the group to WhatsApp |
| `.promote` | `@user` | 🔒 | Promote member to admin |
| `.demote` | `@user` | 🔒 | Demote admin to member |

---

## Business Tools

### BusinessTools
> Check and display WhatsApp Business profiles

| Command | Args | Access | Description |
|---------|------|--------|-------------|
| `.bizprofile` / `.getbizinfo` | `@user` or `number` | 🔒 | Show full Business profile info |
| `.checkbiz` | `@user` or `number` | 🔒 | Quick check if number is a Business account |

---

## Media Tools

### MediaTools
> Download media from messages, convert stickers

| Command | Args | Access | Description |
|---------|------|--------|-------------|
| `.dlimg` | _(reply image)_ | 🔒 | Re-send image from quoted message |
| `.dlvid` | _(reply video)_ | 🔒 | Re-send video from quoted message |
| `.dlaudio` | _(reply audio)_ | 🔒 | Re-send audio from quoted message |
| `.dlfile` | _(reply doc)_ | 🔒 | Re-send document from quoted message |
| `.mediainfo` / `.minfo` | _(reply media)_ | 🔒 | Show media metadata (size, mime, SHA256) |
| `.sticker` | _(reply image/video)_ | 🔒 | Convert image or video to sticker |
| `.unsticker` / `.toimg` | _(reply sticker)_ | 🔒 | Convert sticker back to image |

> Media download uses internal API (`MEDIA_API_URL`, `MEDIA_API_KEY`, `MEDIA_API_KEY2`).

---

## Broadcast Tools

### BroadcastTools
> Broadcast messages, tag all, announce

| Command | Args | Access | Description |
|---------|------|--------|-------------|
| `.broadcast` | `text` | 🔒 | Broadcast to all saved contacts (max 30, 500ms delay) |
| `.groupbroadcast` / `.gbroadcast` | `text` | 🔒 | Send message to all joined groups (max 20, 800ms delay) |
| `.tagall` / `.everyone` | `[text]` | 🔒 | Mention all group members |
| `.tagadmins` / `.pingadmins` | `[text]` | 🔒 | Mention all group admins only |
| `.announce` | `text` | 🔒 | Temporarily unlock group → send → re-lock (admin only) |

---

## Reaction Tools

### ReactTools
> Add and remove emoji reactions on messages

| Command | Args | Access | Description |
|---------|------|--------|-------------|
| `.react` | `emoji` _(reply)_ | 🔒 | React to quoted message with emoji |
| `.react2` | `emoji` _(reply)_ | 🔒 | Alias for `.react` |
| `.unreact` | _(reply)_ | 🔒 | Remove your reaction from quoted message |

---

## Poll Tools

### PollTools
> Create WhatsApp polls in any chat

| Command | Args | Access | Description |
|---------|------|--------|-------------|
| `.poll` | `question \| opt1 \| opt2 [...]` | 🔒 | Create single-choice poll (up to 12 options) |
| `.pollmulti` | `question \| opt1 \| opt2 [...]` | 🔒 | Create multi-select poll |
| `.quickpoll` / `.yesnopoll` | `question` | 🔒 | Quick Yes/No poll |

> Options are separated by `|` pipe character. Max 12 options per poll.

---

## Location Tools

### LocationTools
> Send and extract location messages

| Command | Args | Access | Description |
|---------|------|--------|-------------|
| `.sendloc` / `.loc` | `lat lon [name]` | 🔒 | Send a location pin |
| `.live` / `.liveloc` | `lat lon [seconds]` | 🔒 | Send live location (default 60s) |
| `.getloc` / `.locinfo` | _(reply location)_ | 🔒 | Extract lat/lon + Google Maps link from quoted location |
| `.myloc` | — | 🔒 | Instructions on how to share live location |

---

## Contacts Tools

### ContactsTools
> List, search, block/unblock contacts

| Command | Args | Access | Description |
|---------|------|--------|-------------|
| `.contacts` / `.listcontacts` | — | 🔒 | List all saved contacts (first 20 + total count) |
| `.searchcontact` / `.findcontact` | `query` | 🔒 | Search contacts by name or number |
| `.block` | `@user` or `number` | 🔒 | Block a WhatsApp number |
| `.unblock` | `@user` or `number` | 🔒 | Unblock a number |
| `.blocklist` / `.blocked` | — | 🔒 | Show your full block list |
| `.isblocked` | `@user` or `number` | 🔒 | Check if a number is blocked |
| `.contactinfo` / `.cinfo` | `@user` or `number` | 🔒 | Full contact info (name, push name, biz status) |

---

## Architecture

```
pbx_elite/
├── main.go              — Entry point, event loop
├── pairing.go           — QR/phone pairing
├── config/              — Bot config (prefix, owner JID, sudo list)
├── client/              — whatsmeow client wrapper
├── database/            — Persistent storage
├── mongostore/          — MongoDB session store
├── session/             — Session management
├── web/                 — Web panel
└── modules/             — All command modules (140 files)
    ├── module.go        — HandleMessage dispatcher + helpers
    ├── utils.go         — Reply, ReplyErr, strPtr, etc.
    ├── raidpools.go     — All string pools (3000+ strings)
    ├── raid.go          — Direct raid commands
    ├── replyraid.go     — Auto reply raid system
    └── customraid.go    — Personal raid list
```

---

## Credits

Ported from:
- **PBX Main** — Python/Pyrogram userbot
- **SC Phoenix** — Extended command set
- **PBX UC2** — Additional utilities

Built with [whatsmeow](https://go.mau.fi/whatsmeow) by tulir
