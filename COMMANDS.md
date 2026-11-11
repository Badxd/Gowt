# PBX Elite — Command Reference

> WhatsApp userbot built with [whatsmeow](https://go.mau.fi/whatsmeow)
> **Go-based port of PBX Main (Pyrogram) + SC Phoenix + PBX UC2**

**130 modules · 381 commands**

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
