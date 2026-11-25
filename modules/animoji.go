package modules

import (
	"time"
	"context"
	"fmt"

	"PbxGb/client"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

// animateText sends an animated sequence by editing the same message repeatedly.
// Since WhatsApp userbot can't edit sent messages like Telegram, we simulate
// by sending a message and then sending follow-up frames with a short delay.
func animateSequence(v *events.Message, frames []string, delay time.Duration) error {
	wac := client.Get()
	chatJID := v.Info.Chat

	// Send first frame as reply
	firstText := frames[0]
	firstMsg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: &firstText,
			ContextInfo: &waProto.ContextInfo{
				StanzaID:      &v.Info.ID,
				Participant:   strPtr(v.Info.Sender.String()),
				QuotedMessage: v.Message,
			},
		},
	}
	_, err := wac.SendMessage(context.Background(), chatJID, firstMsg)
	if err != nil {
		return err
	}

	// Send remaining frames with delay (in same chat, no reply context)
	for _, frame := range frames[1:] {
		time.Sleep(delay)
		f := frame
		_, err = wac.SendMessage(context.Background(), chatJID, &waProto.Message{
			Conversation: &f,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func thinkHandler(v *events.Message, args []string) error {
	frames := []string{"🤔", "🧐", "🤔", "🧐", "🤔", "🧐", "🤔🧐"}
	return animateSequence(v, frames, 300*time.Millisecond)
}

func lmaoHandler(v *events.Message, args []string) error {
	frames := []string{"😂", "🤣", "😂", "🤣", "😂🤣", "🤣😂", "😂🤣😂🤣"}
	return animateSequence(v, frames, 300*time.Millisecond)
}

func clockHandler(v *events.Message, args []string) error {
	frames := []string{"🕛", "🕐", "🕑", "🕒", "🕓", "🕔", "🕕", "🕖", "🕗", "🕘", "🕙", "🕚", "🕛"}
	return animateSequence(v, frames, 200*time.Millisecond)
}

func heartAnim(v *events.Message, args []string) error {
	frames := []string{"❤️", "🧡", "💛", "💚", "💙", "💜", "🖤", "🤍", "💖", "❤️🧡💛💚💙💜"}
	return animateSequence(v, frames, 250*time.Millisecond)
}

func muahHandler(v *events.Message, args []string) error {
	frames := []string{"😗", "😙", "😚", "😘", "😘💋", "💋💋💋"}
	return animateSequence(v, frames, 300*time.Millisecond)
}

func fapHandler(v *events.Message, args []string) error {
	frames := []string{"🍆", "✊🏻", "💦", "🍆✊🏻", "✊🏻💦", "🍆✊🏻💦"}
	return animateSequence(v, frames, 300*time.Millisecond)
}

func cryHandler(v *events.Message, args []string) error {
	return ReplyErr(v, "(;´༎ຶД༎ຶ)")
}

func carryHandler(v *events.Message, args []string) error {
	name := ""
	if len(args) > 0 {
		name = fmt.Sprintf("%s\n\n", args[0])
	}
	art := name + `                     ⣤⣶⣶⣶⣦⣤⣄⡀
⠀⠀⠀⠀⠀⣰⣿⣿⣿⣿⣿⣿⣿⣿⣿⣦⡀
⠀⠀⠀⢀⣾⣿⣿⣿⠿⠿⠟⠻⠿⢿⣿⣿⣿⡆
⠀⠀⠀⢰⣿⣿⡿⠂⠀⠀⠀⠀⠀⠀⠈⠉⢻⡇
⠀⠀⠀⠈⠿⣿⣇⣠⠤⠤⠤⢤⣀⣤⠤⠤⣺⡏
⠀⠀⠀⠀⠐⢉⣯⠹⣀⣀⣢⡸⠉⢏⡄⣀⣯⠁
⠀⠀⠀⠀⠡⠀⢹⣆⠀⠀⠀⣀⡀⡰⠀⢠⠖⠂
⠀⠀⠀⠀⠀⠈⠙⣿⣿⠀⠠⠚⢋⡁⠀⡜
⠀⠀⠀⠀⠀⠀⢸⠈⠙⠦⣤⣀⣤⣤⡼⠁
⠀⠀⠀⠀⢀⡌⠀⠀⠀⠀⠉⢏⡉
⠀⠀⠀⣀⣴⣿⣷⣶⣤⣤⣤⣴⣾⣷⣶⣦⡀
⢀⣴⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣄
⠚⠛⠛⠛⠛⠛⠛⠛⠛⠛⠛⠛⠛⠛⠛⠛⠛⠛`
	return ReplyErr(v, art)
}

func dedHandler(v *events.Message, args []string) error {
	name := ""
	if len(args) > 0 {
		name = args[0] + " "
	}
	art := name + `
　　　　　|
　　　　　|
　　　　　|
　　　　　|
　　　　　|
　　　　　|
　　　　　|
　　　　　|
　／￣￣＼|
＜ ´･ 　　|＼
　|　３　 | 丶＼
＜ 、･　　|　　＼
　＼＿＿／∪ _ ∪)
　　　　　 Ｕ Ｕ`
	return ReplyErr(v, art)
}

func init() {
	Register(ModuleInfo{
		Name:        "Animoji",
		Description: "Animated emoji and ASCII art reactions",
		Commands: []CommandInfo{
			{Pattern: "think", Handler: thinkHandler, Sudo: true},
			{Pattern: "lmao", Handler: lmaoHandler, Sudo: true},
			{Pattern: "clock", Handler: clockHandler, Sudo: true},
			{Pattern: "heartan", Handler: heartAnim, Sudo: true},
			{Pattern: "muah", Handler: muahHandler, Sudo: true},
			{Pattern: "fap", Handler: fapHandler, Sudo: true},
			{Pattern: "ccry", Handler: cryHandler, Sudo: true},
			{Pattern: "carry", Handler: carryHandler, Sudo: true},
			{Pattern: "ded", Handler: dedHandler, Sudo: true},
		},
	})
}
