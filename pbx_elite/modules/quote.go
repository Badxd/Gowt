package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"

	"go.mau.fi/whatsmeow/types/events"
)

type quoteData struct {
	Content string `json:"content"`
	Author  string `json:"author"`
}

func fetchRandomQuote() (string, string, error) {
	resp, err := http.Get("https://api.quotable.io/random")
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var q quoteData
	if err := json.Unmarshal(body, &q); err != nil {
		return "", "", err
	}
	return q.Content, q.Author, nil
}

var localQuotes = []struct{ Quote, Author string }{
	{"Code is like humor. When you have to explain it, it's bad.", "Cory House"},
	{"First, solve the problem. Then, write the code.", "John Johnson"},
	{"Experience is the name everyone gives to their mistakes.", "Oscar Wilde"},
	{"In order to be irreplaceable, one must always be different.", "Coco Chanel"},
	{"Java is to JavaScript what car is to Carpet.", "Chris Heilmann"},
	{"Knowledge is power.", "Francis Bacon"},
	{"Talk is cheap. Show me the code.", "Linus Torvalds"},
	{"It works on my machine.", "Every developer ever"},
	{"Sometimes it pays to stay in bed on Monday, rather than spending the rest of the week debugging Monday's code.", "Dan Salomon"},
}

func quoteHandler(v *events.Message, args []string) error {
	content, author, err := fetchRandomQuote()
	if err != nil {
		// fallback to local
		q := localQuotes[rand.Intn(len(localQuotes))]
		content, author = q.Quote, q.Author
	}

	msg := fmt.Sprintf("💬 *Quote:*\n\n_%s_\n\n— *%s*", content, author)
	return ReplyErr(v, msg)
}

func init() {
	Register(ModuleInfo{
		Name:        "Quote",
		Description: "Fetch a random inspirational quote",
		Commands: []CommandInfo{
			{Pattern: "quote", Handler: quoteHandler, Sudo: true},
		},
	})
}
