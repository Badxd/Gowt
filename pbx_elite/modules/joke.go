package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"

	"go.mau.fi/whatsmeow/types/events"
)

type jokeData struct {
	Type  string `json:"type"`
	Setup string `json:"setup"`
	Punchline string `json:"punchline"`
	Joke  string `json:"joke"`
}

func fetchJoke() (string, error) {
	apis := []string{
		"https://official-joke-api.appspot.com/random_joke",
		"https://v2.jokeapi.dev/joke/Programming,Misc?type=twopart&blacklistFlags=nsfw",
	}

	resp, err := http.Get(apis[0])
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var j jokeData
	if err := json.Unmarshal(body, &j); err != nil {
		return "", err
	}

	if j.Joke != "" {
		return j.Joke, nil
	}
	if j.Setup != "" && j.Punchline != "" {
		return j.Setup + "\n\n😂 " + j.Punchline, nil
	}
	return "", fmt.Errorf("no joke")
}

var localJokes = []string{
	"Why do programmers prefer dark mode?\nBecause light attracts bugs! 🐛",
	"Why did the developer go broke?\nBecause they used up all their cache! 💸",
	"A SQL query walks into a bar, walks up to two tables and asks...\n'Can I join you?' 😂",
	"How many programmers does it take to change a light bulb?\nNone — it's a hardware problem! 💡",
	"Why do Java developers wear glasses?\nBecause they don't C#! 👓",
	"!false — It's funny because it's true 😂",
	"Debugging: Being the detective in a crime movie where you're also the murderer 🔍",
}


func init() {
	Register(ModuleInfo{
		Name:        "Joke",
		Description: "Get a random joke",
		Commands: []CommandInfo{
		},
	})
}
