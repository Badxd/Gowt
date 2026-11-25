package modules

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// ── Hangman game ──────────────────────────────────────────────────────────────

var hangmanWords = []string{
	"programming", "computer", "javascript", "golang", "python", "database",
	"algorithm", "variable", "function", "compiler", "operating", "network",
	"security", "encryption", "protocol", "interface", "keyboard", "monitor",
	"processor", "bluetooth", "wireless", "software", "hardware", "firmware",
	"bandwidth", "latency", "debugging", "recursion", "iteration", "framework",
	"whatsapp", "telegram", "message", "notification", "developer", "terminal",
}

type hangmanGame struct {
	word      string
	guessed   map[rune]bool
	wrong     int
	maxWrong  int
	startedBy string
}

var (
	hangmanMu    sync.Mutex
	hangmanGames = make(map[string]*hangmanGame) // chatJID → game
)

var hangmanArt = []string{
	"```\n  +---+\n  |   |\n      |\n      |\n      |\n      |\n=========```",
	"```\n  +---+\n  |   |\n  O   |\n      |\n      |\n      |\n=========```",
	"```\n  +---+\n  |   |\n  O   |\n  |   |\n      |\n      |\n=========```",
	"```\n  +---+\n  |   |\n  O   |\n /|   |\n      |\n      |\n=========```",
	"```\n  +---+\n  |   |\n  O   |\n /|\\  |\n      |\n      |\n=========```",
	"```\n  +---+\n  |   |\n  O   |\n /|\\  |\n /    |\n      |\n=========```",
	"```\n  +---+\n  |   |\n  O   |\n /|\\  |\n / \\  |\n      |\n=========```",
}

func (g *hangmanGame) display() string {
	// Build word display
	var wordDisplay []string
	for _, ch := range g.word {
		if g.guessed[ch] {
			wordDisplay = append(wordDisplay, string(ch))
		} else {
			wordDisplay = append(wordDisplay, "_")
		}
	}

	// Wrong guesses
	var wrong []string
	for ch, guessed := range g.guessed {
		if !strings.ContainsRune(g.word, ch) && guessed {
			wrong = append(wrong, string(ch))
		}
	}

	art := hangmanArt[g.wrong]
	if g.wrong >= len(hangmanArt) {
		art = hangmanArt[len(hangmanArt)-1]
	}

	return fmt.Sprintf("%s\n\n🔤 Word: %s\n❌ Wrong: %s (%d/%d)",
		art,
		strings.Join(wordDisplay, " "),
		strings.Join(wrong, ", "),
		g.wrong, g.maxWrong,
	)
}

func (g *hangmanGame) isWon() bool {
	for _, ch := range g.word {
		if !g.guessed[ch] {
			return false
		}
	}
	return true
}

// .hm start — start a hangman game
// .hm <letter> — guess a letter
// .hm stop — end the game
func hangmanHandler(v *events.Message, args []string) error {
	chat := v.Info.Chat.String()
	sender := v.Info.Sender.String()

	hangmanMu.Lock()
	defer hangmanMu.Unlock()

	if len(args) == 0 {
		return ReplyErr(v, "Usage:\n.hm start — start game\n.hm <letter> — guess a letter\n.hm stop — quit game\n.hm hint — reveal a letter")
	}

	switch strings.ToLower(args[0]) {
	case "start":
		if _, exists := hangmanGames[chat]; exists {
			return ReplyErr(v, "❌ A game is already running! Use .hm <letter> to guess.")
		}
		src := rand.NewSource(time.Now().UnixNano())
		r := rand.New(src)
		word := hangmanWords[r.Intn(len(hangmanWords))]

		game := &hangmanGame{
			word:      word,
			guessed:   make(map[rune]bool),
			wrong:     0,
			maxWrong:  6,
			startedBy: sender,
		}
		hangmanGames[chat] = game
		return ReplyErr(v, "🎮 *Hangman started!*\n\n"+game.display()+"\n\nGuess a letter: .hm <letter>")

	case "stop":
		game, exists := hangmanGames[chat]
		if !exists {
			return ReplyErr(v, "❌ No active game.")
		}
		delete(hangmanGames, chat)
		return ReplyErr(v, fmt.Sprintf("🛑 Game stopped. The word was: *%s*", game.word))

	case "hint":
		game, exists := hangmanGames[chat]
		if !exists {
			return ReplyErr(v, "❌ No active game. Use .hm start")
		}
		// Reveal an unguessed letter
		for _, ch := range game.word {
			if !game.guessed[ch] {
				game.guessed[ch] = true
				game.wrong++ // cost one wrong for a hint
				return ReplyErr(v, fmt.Sprintf("💡 Hint: revealed *%s*\n\n%s", string(ch), game.display()))
			}
		}
		return ReplyErr(v, "All letters already guessed!")

	default:
		// Guess a letter
		game, exists := hangmanGames[chat]
		if !exists {
			return ReplyErr(v, "❌ No active game. Use .hm start")
		}

		guess := []rune(strings.ToLower(args[0]))
		if len(guess) != 1 {
			return ReplyErr(v, "❌ Guess one letter at a time.")
		}
		letter := guess[0]

		if game.guessed[letter] {
			return ReplyErr(v, fmt.Sprintf("⚠️ You already guessed '%s'", string(letter)))
		}
		game.guessed[letter] = true

		if !strings.ContainsRune(game.word, letter) {
			game.wrong++
		}

		display := game.display()

		if game.isWon() {
			delete(hangmanGames, chat)
			return ReplyErr(v, fmt.Sprintf("🎉 *You won!* The word was: *%s*\n\n%s", game.word, display))
		}
		if game.wrong >= game.maxWrong {
			delete(hangmanGames, chat)
			return ReplyErr(v, fmt.Sprintf("💀 *Game over!* The word was: *%s*\n\n%s", game.word, display))
		}
		return ReplyErr(v, display)
	}
}

func init() {
	Register(ModuleInfo{
		Name:        "Hangman",
		Description: "Play hangman in WhatsApp",
		Commands: []CommandInfo{
			{Pattern: "hm", Handler: hangmanHandler, Sudo: true},
		},
	})
}
