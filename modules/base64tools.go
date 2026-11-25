package modules

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

func b64EncodeHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: `.b64en <text>`")
	}
	text := strings.Join(args, " ")
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	return ReplyErr(v, fmt.Sprintf("🔐 *Base64 Encoded:*\n\n`%s`", encoded))
}

func b64DecodeHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: `.b64de <encoded text>`")
	}
	text := strings.Join(args, " ")
	decoded, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		// try URL encoding
		decoded, err = base64.URLEncoding.DecodeString(text)
		if err != nil {
			return ReplyErr(v, "❌ Invalid base64 string.")
		}
	}
	return ReplyErr(v, fmt.Sprintf("🔓 *Base64 Decoded:*\n\n`%s`", string(decoded)))
}

func hashHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: `.hash <text>`")
	}
	text := strings.Join(args, " ")
	data := []byte(text)

	md5sum := fmt.Sprintf("%x", md5.Sum(data))
	sha1sum := fmt.Sprintf("%x", sha1.Sum(data))
	sha256sum := fmt.Sprintf("%x", sha256.Sum256(data))
	sha512sum := fmt.Sprintf("%x", sha512.Sum512(data))

	return ReplyErr(v, fmt.Sprintf(
		"🔑 *Hash Values*\n\nText: `%s`\n\n"+
			"*MD5:*\n`%s`\n\n"+
			"*SHA1:*\n`%s`\n\n"+
			"*SHA256:*\n`%s`\n\n"+
			"*SHA512:*\n`%s`",
		text, md5sum, sha1sum, sha256sum, sha512sum,
	))
}

func init() {
	Register(ModuleInfo{
		Name:        "Base64 & Hash",
		Description: "Base64 encode/decode and hash text",
		Commands: []CommandInfo{
			{Pattern: "b64en", Handler: b64EncodeHandler, Sudo: true},
			{Pattern: "b64de", Handler: b64DecodeHandler, Sudo: true},
			{Pattern: "hash", Handler: hashHandler, Sudo: true},
		},
	})
}
