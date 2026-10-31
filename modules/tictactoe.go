package modules

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"PbxGb/client"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

type tttGame struct {
	board     [9]string // "", "X", "O"
	playerX   string    // JID string
	playerO   string    // JID string, "" while waiting for an opponent
	turn      string    // JID string of whoever moves next
	startedAt time.Time
}

var (
	tttMu    sync.Mutex
	tttGames = make(map[string]*tttGame) // key: chat JID string
)

var tttWinLines = [8][3]int{
	{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // rows
	{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // columns
	{0, 4, 8}, {2, 4, 6}, // diagonals
}

func tttWinner(board [9]string) string {
	for _, line := range tttWinLines {
		a, b, c := board[line[0]], board[line[1]], board[line[2]]
		if a != "" && a == b && b == c {
			return a
		}
	}
	return ""
}

var tttSymbolEmoji = map[string]string{
	"": "", "X": "❎", "O": "⭕",
}
var tttNumberEmoji = []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣"}

func renderTTTBoard(g *tttGame) string {
	var b strings.Builder
	for i, cell := range g.board {
		if cell == "" {
			b.WriteString(tttNumberEmoji[i])
		} else {
			b.WriteString(tttSymbolEmoji[cell])
		}
		if (i+1)%3 == 0 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func tttSendBoard(v *events.Message, g *tttGame, header string) {
	text := fmt.Sprintf(
		"🎮 *TicTacToe*\n\n%s\n\n%s\n▢ Player ❎: @%s\n▢ Player ⭕: @%s",
		header, renderTTTBoard(g),
		strings.Split(g.playerX, "@")[0], strings.Split(g.playerO, "@")[0],
	)
	wac := client.Get()
	mentions := []string{g.playerX}
	if g.playerO != "" {
		mentions = append(mentions, g.playerO)
	}
	_, _ = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text:        &text,
			ContextInfo: &waProto.ContextInfo{MentionedJID: mentions},
		},
	})
}

// .ttt [room name] — start or join a game. Open to anyone (Sudo: false),
// unlike most commands here, since a 2-player game only makes sense if
// both players can actually trigger it.
func tttHandler(v *events.Message, args []string) error {
	if !isGroup(v) {
		return ReplyErr(v, "❌ TicTacToe only works in groups.")
	}

	key := v.Info.Chat.String()
	sender := v.Info.Sender.String()

	tttMu.Lock()
	defer tttMu.Unlock()

	g, exists := tttGames[key]

	if exists && g.playerO != "" {
		if sender == g.playerX || sender == g.playerO {
			return ReplyErr(v, "❌ You're already in a game here. Type *surrender* to quit.")
		}
		return ReplyErr(v, "❌ A game is already running here. Wait for it to finish.")
	}

	if exists && g.playerO == "" {
		if sender == g.playerX {
			return ReplyErr(v, "⏳ Still waiting for someone else to join with *.ttt*.")
		}
		g.playerO = sender
		g.turn = g.playerX
		tttSendBoard(v, g, fmt.Sprintf("Game started! @%s vs @%s\nTurn: @%s (❎)\n\nType a number 1-9 to play.",
			strings.Split(g.playerX, "@")[0], strings.Split(g.playerO, "@")[0], strings.Split(g.turn, "@")[0]))
		return nil
	}

	g = &tttGame{playerX: sender, turn: sender, startedAt: time.Now()}
	tttGames[key] = g
	return ReplyErr(v, "⏳ *Waiting for opponent*\nAnyone in this group can type *.ttt* to join!")
}

// HandleTTTMove — call on every plain-text message (from main.go). Only
// acts if the sender is a player in an active game in this chat, so it's
// safe to call unconditionally regardless of who's talking.
func HandleTTTMove(v *events.Message) {
	if !isGroup(v) {
		return
	}
	key := v.Info.Chat.String()

	tttMu.Lock()
	g, exists := tttGames[key]
	tttMu.Unlock()
	if !exists || g.playerO == "" {
		return
	}

	sender := v.Info.Sender.String()
	if sender != g.playerX && sender != g.playerO {
		return
	}

	text := strings.ToLower(strings.TrimSpace(extractText(v)))
	if text == "" {
		return
	}

	isSurrender := text == "surrender" || text == "give up"
	if !isSurrender {
		if _, err := strconv.Atoi(text); err != nil {
			return
		}
	}

	tttMu.Lock()
	defer tttMu.Unlock()

	// Re-check under lock in case the game just ended.
	g, exists = tttGames[key]
	if !exists || g.playerO == "" {
		return
	}

	if isSurrender {
		winner := g.playerO
		if sender == g.playerO {
			winner = g.playerX
		}
		delete(tttGames, key)
		wac := client.Get()
		text := fmt.Sprintf("🏳️ @%s surrendered! @%s wins!",
			strings.Split(sender, "@")[0], strings.Split(winner, "@")[0])
		_, _ = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text:        &text,
				ContextInfo: &waProto.ContextInfo{MentionedJID: []string{sender, winner}},
			},
		})
		return
	}

	if sender != g.turn {
		_ = ReplyErr(v, "❌ Not your turn!")
		return
	}

	n, _ := strconv.Atoi(text)
	idx := n - 1
	if idx < 0 || idx > 8 || g.board[idx] != "" {
		_ = ReplyErr(v, "❌ Invalid move — pick an empty box (1-9).")
		return
	}

	symbol := "X"
	if sender == g.playerO {
		symbol = "O"
	}
	g.board[idx] = symbol

	winnerSym := tttWinner(g.board)
	full := true
	for _, c := range g.board {
		if c == "" {
			full = false
			break
		}
	}

	switch {
	case winnerSym != "":
		winner := g.playerX
		if winnerSym == "O" {
			winner = g.playerO
		}
		delete(tttGames, key)
		tttSendBoard(v, g, fmt.Sprintf("🎉 @%s wins!", strings.Split(winner, "@")[0]))
	case full:
		delete(tttGames, key)
		tttSendBoard(v, g, "🤝 It's a draw!")
	default:
		if g.turn == g.playerX {
			g.turn = g.playerO
		} else {
			g.turn = g.playerX
		}
		nextSym := "❎"
		if g.turn == g.playerO {
			nextSym = "⭕"
		}
		tttSendBoard(v, g, fmt.Sprintf("Turn: @%s (%s)", strings.Split(g.turn, "@")[0], nextSym))
	}
}

func init() {
	Register(ModuleInfo{
		Name:        "TicTacToe",
		Description: "2-player TicTacToe, open to everyone in the group",
		Commands: []CommandInfo{
			{Pattern: "ttt", Handler: tttHandler, Sudo: false},
		},
	})
}
