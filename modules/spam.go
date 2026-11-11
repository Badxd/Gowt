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
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	spamMu     sync.Mutex
	spamCancel = make(map[string]context.CancelFunc)
)

func cancelSpam(jid string) bool {
	spamMu.Lock()
	defer spamMu.Unlock()
	if cancel, ok := spamCancel[jid]; ok {
		cancel()
		delete(spamCancel, jid)
		return true
	}
	return false
}

// .bigspam <count> <message>
func bigSpamHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		return ReplyErr(v, "Usage: .bigspam <count> <message>")
	}
	count, err := strconv.Atoi(args[0])
	if err != nil || count <= 0 {
		return ReplyErr(v, "❌ Invalid count.")
	}
	if count > 100 {
		count = 100
	}
	text := strings.Join(args[1:], " ")
	jidStr := v.Info.Chat.String()
	chat := v.Info.Chat
	wac := client.Get()

	ctx, cancel := context.WithCancel(context.Background())
	spamMu.Lock()
	if prev, ok := spamCancel[jidStr]; ok {
		prev()
	}
	spamCancel[jidStr] = cancel
	spamMu.Unlock()

	go func() {
		for i := 0; i < count; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				t := text
				wac.SendMessage(ctx, chat, &waProto.Message{Conversation: &t})
				time.Sleep(300 * time.Millisecond)
			}
		}
		spamMu.Lock()
		delete(spamCancel, jidStr)
		spamMu.Unlock()
	}()

	return ReplyErr(v, fmt.Sprintf("🔁 Spamming *%d* times...\nUse .stopbigspam to stop.", count))
}

func stopBigSpamHandler(v *events.Message, args []string) error {
	if cancelSpam(v.Info.Chat.String()) {
		return ReplyErr(v, "✅ Spam stopped.")
	}
	return ReplyErr(v, "ℹ️ No active spam session.")
}

var galiList = []string{
	"chutiye 🤬", "bhosdike 😤", "madarchod 💀",
	"ullu ke patthe 🐦", "haramkhor 😡", "nalayak 🤦",
	"gadhe 🐴", "bekaar 😑", "jhoothe 🤡",
}

func galiSpamHandler(v *events.Message, args []string) error {
	count := 10
	if len(args) > 0 {
		if n, err := strconv.Atoi(args[0]); err == nil && n > 0 {
			count = n
		}
	}
	if count > 50 {
		count = 50
	}
	jidStr := v.Info.Chat.String()
	chat := v.Info.Chat
	wac := client.Get()

	ctx, cancel := context.WithCancel(context.Background())
	spamMu.Lock()
	if prev, ok := spamCancel[jidStr]; ok {
		prev()
	}
	spamCancel[jidStr] = cancel
	spamMu.Unlock()

	go func() {
		for i := 0; i < count; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				t := galiList[rand.Intn(len(galiList))]
				wac.SendMessage(ctx, chat, &waProto.Message{Conversation: &t})
				time.Sleep(300 * time.Millisecond)
			}
		}
		spamMu.Lock()
		delete(spamCancel, jidStr)
		spamMu.Unlock()
	}()

	return ReplyErr(v, fmt.Sprintf("🔁 Gali spam started (%d)...\nUse .stopgalispam to stop.", count))
}

func stopGaliSpamHandler(v *events.Message, args []string) error {
	if cancelSpam(v.Info.Chat.String()) {
		return ReplyErr(v, "✅ Gali spam stopped.")
	}
	return ReplyErr(v, "ℹ️ No active gali spam.")
}

// .multispam <count> <jid1,jid2,...> <message>
func multiSpamHandler(v *events.Message, args []string) error {
	if len(args) < 3 {
		return ReplyErr(v, "Usage: .multispam <count> <jid1,jid2> <message>")
	}
	count, err := strconv.Atoi(args[0])
	if err != nil || count <= 0 {
		return ReplyErr(v, "❌ Invalid count.")
	}
	if count > 50 {
		count = 50
	}
	jids := strings.Split(args[1], ",")
	text := strings.Join(args[2:], " ")
	wac := client.Get()

	var targets []types.JID
	for _, raw := range jids {
		jid, err := types.ParseJID(strings.TrimSpace(raw))
		if err == nil {
			targets = append(targets, jid)
		}
	}
	if len(targets) == 0 {
		return ReplyErr(v, "❌ No valid JIDs parsed.")
	}

	go func() {
		for i := 0; i < count; i++ {
			for _, jid := range targets {
				t := text
				wac.SendMessage(context.Background(), jid, &waProto.Message{Conversation: &t})
				time.Sleep(300 * time.Millisecond)
			}
		}
	}()

	return ReplyErr(v, fmt.Sprintf("🔁 Multi-spam started: %d×%d targets", count, len(targets)))
}

func init() {
	Register(ModuleInfo{
		Name:        "Spam",
		Description: "Spam and flood tools (bigspam, galispam, multispam)",
		Commands: []CommandInfo{
			{Pattern: "bigspam", Handler: bigSpamHandler, Sudo: true},
			{Pattern: "stopbigspam", Handler: stopBigSpamHandler, Sudo: true},
			{Pattern: "galispam", Handler: galiSpamHandler, Sudo: true},
			{Pattern: "stopgalispam", Handler: stopGaliSpamHandler, Sudo: true},
			{Pattern: "multispam", Handler: multiSpamHandler, Sudo: true},
		},
	})
}
