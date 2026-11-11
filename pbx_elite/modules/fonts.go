package modules

import (
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

// ─── Font character maps ───────────────────────────────────────────────────

const fontDefault = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var fontMaps = map[string]string{
	"smallcap":    "ᴀʙᴄᴅᴇғɢʜɪᴊᴋʟᴍɴᴏᴘϙʀsᴛᴜᴠᴡxʏᴢABCDEFGHIJKLMNOPQRSTUVWXYZ",
	"monospace":   "𝚊𝚋𝚌𝚍𝚎𝚏𝚐𝚑𝚒𝚓𝚔𝚕𝚖𝚗𝚘𝚙𝚚𝚛𝚜𝚝𝚞𝚟𝚠𝚡𝚢𝚣𝙰𝙱𝙲𝙳𝙴𝙵𝙶𝙷𝙸𝙹𝙺𝙻𝙼𝙽𝙾𝙿𝚀𝚁𝚂𝚃𝚄𝚅𝚆𝚇𝚈𝚉",
	"outline":     "𝕒𝕓𝕔𝕕𝕖𝕗𝕘𝕙𝕚𝕛𝕜𝕝𝕞𝕟𝕠𝕡𝕢𝕣𝕤𝕥𝕦𝕧𝕨𝕩𝕪𝕫𝔸𝔹ℂ𝔻𝔼𝔽𝔾ℍ𝕀𝕁𝕂𝕃𝕄ℕ𝕆ℙℚℝ𝕊𝕋𝕌𝕍𝕎𝕏𝕐ℤ",
	"script":      "𝒶𝒷𝒸𝒹𝑒𝒻𝑔𝒽𝒾𝒿𝓀𝓁𝓂𝓃𝑜𝓅𝓆𝓇𝓈𝓉𝓊𝓋𝓌𝓍𝓎𝓏𝒜ℬ𝒞𝒟ℰℱ𝒢ℋℐ𝒥𝒦ℒℳ𝒩𝒪𝒫𝒬ℛ𝒮𝒯𝒰𝒱𝒲𝒳𝒴𝒵",
	"blackbubbles": "🅐🅑🅒🅓🅔🅕🅖🅗🅘🅙🅚🅛🅜🅝🅞🅟🅠🅡🅢🅣🅤🅥🅦🅧🅨🅩🅐🅑🅒🅓🅔🅕🅖🅗🅘🅙🅚🅛🅜🅝🅞🅟🅠🅡🅢🅣🅤🅥🅦🅧🅨🅩",
	"bubbles":     "ⓐⓑⓒⓓⓔⓕⓖⓗⓘⓙⓚⓛⓜⓝⓞⓟⓠⓡⓢⓣⓤⓥⓦⓧⓨⓩⒶⒷⒸⒹⒺⒻⒼⒽⒾⒿⓀⓁⓂⓃⓄⓅⓆⓇⓈⓉⓊⓋⓌⓍⓎⓏ",
	"bold":        "𝗮𝗯𝗰𝗱𝗲𝗳𝗴𝗵𝗶𝗷𝗸𝗹𝗺𝗻𝗼𝗽𝗾𝗿𝘀𝘁𝘂𝘃𝘄𝘅𝘆𝘇𝗔𝗕𝗖𝗗𝗘𝗙𝗚𝗛𝗜𝗝𝗞𝗟𝗠𝗡𝗢𝗣𝗤𝗥𝗦𝗧𝗨𝗩𝗪𝗫𝗬𝗭",
	"bolditalic":  "𝙖𝙗𝙘𝙙𝙚𝙛𝙜𝙝𝙞𝙟𝙠𝙡𝙢𝙣𝙤𝙥𝙦𝙧𝙨𝙩𝙪𝙫𝙬𝙭𝙮𝙯𝘼𝘽𝘾𝘿𝙀𝙁𝙂𝙃𝙄𝙅𝙆𝙇𝙈𝙉𝙊𝙋𝙌𝙍𝙎𝙏𝙐𝙑𝙒𝙓𝙔𝙕",
}

func applyFont(text, fontName string) string {
	mapped, ok := fontMaps[fontName]
	if !ok {
		return text
	}
	// build rune slices so index matches
	src := []rune(fontDefault)
	dst := []rune(mapped)
	out := []rune(text)
	for i, ch := range out {
		for j, s := range src {
			if ch == s && j < len(dst) {
				out[i] = dst[j]
				break
			}
		}
	}
	return string(out)
}

func fontHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .font <fontname> <text> OR reply to a message with .font <fontname>")
	}
	fontName := strings.ToLower(args[0])
	if _, ok := fontMaps[fontName]; !ok {
		return ReplyErr(v, "❌ Unknown font. Use *.fontlist* to see available fonts.")
	}

	var text string
	if len(args) > 1 {
		text = strings.Join(args[1:], " ")
	} else {
		// try quoted message
		q := GetQuotedMessage(v)
		if q != nil {
			if q.Conversation != nil {
				text = *q.Conversation
			} else if q.ExtendedTextMessage != nil {
				text = q.ExtendedTextMessage.GetText()
			}
		}
	}
	if text == "" {
		return ReplyErr(v, "❌ Provide text or reply to a message.")
	}
	return ReplyErr(v, applyFont(text, fontName))
}

func fontListHandler(v *events.Message, args []string) error {
	return ReplyErr(v,
		"*📝 Available Fonts:*\n\n"+
			"• smallcap\n"+
			"• monospace\n"+
			"• outline\n"+
			"• script\n"+
			"• blackbubbles\n"+
			"• bubbles\n"+
			"• bold\n"+
			"• bolditalic\n\n"+
			"Usage: `.font <name> <text>` or reply to a msg with `.font <name>`",
	)
}

func init() {
	Register(ModuleInfo{
		Name:        "Fonts",
		Description: "Convert text to fancy Unicode fonts",
		Commands: []CommandInfo{
			{Pattern: "font", Handler: fontHandler, Sudo: true},
			{Pattern: "fontlist", Handler: fontListHandler, Sudo: true},
		},
	})
}
