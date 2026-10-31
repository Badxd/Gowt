# PBX Elite — WhatsApp Userbot (Go)

📢 Updates: https://t.me/PBX_UPDATE · 💬 Support: https://t.me/PBXCHATS · 👑 Owner: https://t.me/BADMUNDAXD

Single-session WhatsApp userbot with a PBX Elite style **pairing website**,
PBX Elite menu / ping / alive, animations, AFK and PM permit.

## Deploy (Heroku)

1. Create the app, set Config Vars:
   * `MONGO_URI` — MongoDB connection string
   * `OWNER_JID` — your number, digits only (e.g. `919646191598`)
   * `PREFIX` — optional (default `.`)
   * `PAIR_KEY` — optional access key for the pair page
   * `GOFLAGS` = `-mod=mod` (lets the first Heroku build download the Go modules)
2. Deploy. Heroku Procfile is `web: PbxGb` (the website needs a **web** dyno).
3. Open your app URL (domain) → the **pair page** appears (white page, black text).
4. Type your number → copy the code → WhatsApp → Linked devices → *Link with phone number*.
5. When linking succeeds the userbot goes live on that number and the same URL
   now shows the **dashboard**. The WhatsApp session (keys, sessions, prekeys, app-state) lives in **MongoDB** — there is no SQLite any more, so Heroku restarts never lose it.

Only **one** session can exist. After it is linked the pair endpoint is closed.
If you remove the device from your phone, the bot clears the session and the
pair page comes back.

`SESSION` env is only used to import an old session once (when MongoDB has none).

## Commands

Send `.menu` (or `.help`) in any chat. Main groups: general (`ping`, `alive`, `menu`),
animations (`anim`, `animlist`, `randomanim` + PBX Elite animations like `love`, `crush`,
`dino`, `brain`, `hypo` …), `afk`, PM permit (`pmsec`, `approve`, `block` …), downloads,
utility, fun, media, protection, group tools, sudo.

## Notes

* AFK clears itself when you send any message from your phone, and also replies when
  you are @mentioned / replied to in a group.
* PM permit auto-approves a chat when you write to it first, and works with `@lid` senders.
* Animations are capped (max 8 frames, 700 ms, 4 s cooldown) to avoid WhatsApp bans.

## Library

`whatsmeow` (https://github.com/tulir/whatsmeow, latest main) is shipped inside this
zip in `./whatsmeow` and wired with a `replace` in `go.mod`. The MongoDB store that
replaces `sqlstore` is in `./mongostore`.

Before the first deploy run once (needs Go 1.26+):

    go mod tidy && go build ./...

and commit the generated `go.sum`.
