package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

type udResponse struct {
	List []struct {
		Word       string `json:"word"`
		Definition string `json:"definition"`
		Example    string `json:"example"`
		ThumbsUp   int    `json:"thumbs_up"`
		ThumbsDown int    `json:"thumbs_down"`
	} `json:"list"`
}

func udHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .ud <word or phrase>")
	}
	query := strings.Join(args, " ")
	encoded := url.QueryEscape(query)
	apiURL := "https://api.urbandictionary.com/v0/define?term=" + encoded

	resp, err := http.Get(apiURL)
	if err != nil {
		return ReplyErr(v, "❌ Request failed: "+err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ReplyErr(v, "❌ Read error")
	}

	var data udResponse
	if err := json.Unmarshal(body, &data); err != nil || len(data.List) == 0 {
		return ReplyErr(v, fmt.Sprintf("❌ No definition found for: *%s*", query))
	}

	entry := data.List[0]
	// Clean up [] brackets from UD formatting
	def := strings.ReplaceAll(entry.Definition, "[", "")
	def = strings.ReplaceAll(def, "]", "")
	ex := strings.ReplaceAll(entry.Example, "[", "")
	ex = strings.ReplaceAll(ex, "]", "")

	msg := fmt.Sprintf(`📖 *Urban Dictionary: %s*

*Definition:*
%s

*Example:*
_%s_

👍 %d | 👎 %d`,
		entry.Word, def, ex,
		entry.ThumbsUp, entry.ThumbsDown,
	)

	return ReplyErr(v, msg)
}

func init() {
	Register(ModuleInfo{
		Name:        "UrbanDict",
		Description: "Look up words on Urban Dictionary",
		Commands: []CommandInfo{
			{Pattern: "ud", Handler: udHandler, Sudo: true},
		},
	})
}
