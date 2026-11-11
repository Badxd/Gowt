package modules

import (
	"context"
	"fmt"
	"log"
	"sync"

	"PbxGb/client"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	callRejectEnabled = true
	callRejectMu      sync.RWMutex
	callRejectMsg     = "🚫 My owner is busy right now.\nPlease wait, they will get back to you soon.\n\n*- PBX Elite Auto Reply*"
)

func HandleCall(v *events.CallOffer) {
	callRejectMu.RLock()
	enabled := callRejectEnabled
	callRejectMu.RUnlock()
	if !enabled {
		return
	}

	wac := client.Get()
	if err := wac.RejectCall(context.Background(), v.CallCreator, v.CallID); err != nil {
		log.Printf("❌ Call reject error: %v", err)
	} else {
		log.Printf("📵 Call rejected from: %s", v.CallCreator.String())
	}

	callRejectMu.RLock()
	msg := callRejectMsg
	callRejectMu.RUnlock()

	callerJID := types.NewJID(v.CallCreator.User, types.DefaultUserServer)
	_, _ = wac.SendMessage(context.Background(), callerJID, &waProto.Message{
		Conversation: &msg,
	})
}

func callRejectToggleHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		callRejectMu.RLock()
		status := callRejectEnabled
		callRejectMu.RUnlock()
		state := "🟢 ON"
		if !status {
			state = "🔴 OFF"
		}
		return ReplyErr(v, fmt.Sprintf("📵 Auto Call Reject: *%s*\n\nWhen a call is rejected, the caller receives this message:\n_%s_", state, callRejectMsg))
	}
	switch args[0] {
	case "on":
		callRejectMu.Lock()
		callRejectEnabled = true
		callRejectMu.Unlock()
		return ReplyErr(v, "✅ Auto Call Reject *ON*\nAll incoming calls will now be automatically rejected and the caller will receive a busy message.")
	case "off":
		callRejectMu.Lock()
		callRejectEnabled = false
		callRejectMu.Unlock()
		return ReplyErr(v, "🔴 Auto Call Reject *OFF*")
	}
	return ReplyErr(v, "❌ Use: .callreject on / off")
}

func callMsgHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		callRejectMu.RLock()
		msg := callRejectMsg
		callRejectMu.RUnlock()
		return ReplyErr(v, fmt.Sprintf("📝 Current call reject message:\n\n%s", msg))
	}
	newMsg := GetArgs(v)
	if newMsg == "" {
		return ReplyErr(v, "❌ Provide a message: .callmsg My owner is busy!")
	}
	callRejectMu.Lock()
	callRejectMsg = newMsg
	callRejectMu.Unlock()
	return ReplyErr(v, fmt.Sprintf("✅ Call reject message updated:\n\n%s", newMsg))
}

func init() {
	Register(ModuleInfo{
		Name:        "CallReject",
		Description: "Auto reject calls + busy message to caller",
		Commands: []CommandInfo{
			{Pattern: "callreject", Handler: callRejectToggleHandler, Sudo: true},
			{Pattern: "callmsg",   Handler: callMsgHandler,           Sudo: true},
		},
	})
}
