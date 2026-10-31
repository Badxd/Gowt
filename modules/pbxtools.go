package modules

// PBX Elite offline toolkit — ported from PBX Elite's offlineCommands.js /
// advancedCommands.js / moreKnightCommands.js. Pure Go, no external API.

import (
	"fmt"
	"math/big"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"PbxGb/config"

	"go.mau.fi/whatsmeow/types/events"
)

// ── helpers ─────────────────────────────────────────────────────────────────

func toolBox(title, body string) string {
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	var sb strings.Builder
	sb.WriteString("┌───⭓ " + smallCaps(title) + " ⭓───\n")
	for _, l := range lines {
		sb.WriteString("│ " + l + "\n")
	}
	sb.WriteString(boxEnd)
	return sb.String()
}

// toolInput = text after the command, or the text of the replied-to message.
func toolInput(v *events.Message) string {
	if in := strings.TrimSpace(GetArgs(v)); in != "" {
		return in
	}
	if q := GetQuotedMessage(v); q != nil {
		if q.GetConversation() != "" {
			return q.GetConversation()
		}
		if e := q.GetExtendedTextMessage(); e != nil {
			return e.GetText()
		}
	}
	return ""
}

type tool struct {
	name  string
	title string
	usage string
	fn    func(in string, v *events.Message) (string, error)
}

func (t tool) handler() func(v *events.Message, args []string) error {
	return func(v *events.Message, args []string) error {
		in := toolInput(v)
		if in == "" && t.usage != "" {
			return ReplyErr(v, "ℹ️ *Usage:* `"+config.C.Prefix+t.usage+"`")
		}
		out, err := t.fn(in, v)
		if err != nil {
			return ReplyErr(v, "❌ "+err.Error())
		}
		return ReplyErr(v, toolBox(t.title, out))
	}
}

func pick(list []string) string { return list[rand.Intn(len(list))] }

func parseFloats(in string, n int) ([]float64, error) {
	f := strings.Fields(strings.ReplaceAll(in, ",", " "))
	if len(f) < n {
		return nil, fmt.Errorf("need %d number(s)", n)
	}
	out := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		x, err := strconv.ParseFloat(f[i], 64)
		if err != nil {
			return nil, fmt.Errorf("\"%s\" is not a number", f[i])
		}
		out = append(out, x)
	}
	return out, nil
}

func parseInt(s string) (int64, error) {
	x, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("\"%s\" is not a whole number", s)
	}
	return x, nil
}

// ── text tools ──────────────────────────────────────────────────────────────

var morseEnc = map[rune]string{
	'A': ".-", 'B': "-...", 'C': "-.-.", 'D': "-..", 'E': ".", 'F': "..-.", 'G': "--.", 'H': "....", 'I': "..",
	'J': ".---", 'K': "-.-", 'L': ".-..", 'M': "--", 'N': "-.", 'O': "---", 'P': ".--.", 'Q': "--.-", 'R': ".-.",
	'S': "...", 'T': "-", 'U': "..-", 'V': "...-", 'W': ".--", 'X': "-..-", 'Y': "-.--", 'Z': "--..",
	'0': "-----", '1': ".----", '2': "..---", '3': "...--", '4': "....-", '5': ".....",
	'6': "-....", '7': "--...", '8': "---..", '9': "----.",
	'.': ".-.-.-", ',': "--..--", '?': "..--..", '!': "-.-.--",
}

func morseTool(in string, _ *events.Message) (string, error) {
	isMorse := strings.Trim(in, ".-/ \n") == ""
	if isMorse {
		dec := map[string]rune{}
		for k, v := range morseEnc {
			dec[v] = k
		}
		var sb strings.Builder
		for _, w := range strings.Split(strings.TrimSpace(in), "/") {
			for _, code := range strings.Fields(w) {
				if r, ok := dec[code]; ok {
					sb.WriteRune(r)
				} else {
					sb.WriteRune('?')
				}
			}
			sb.WriteRune(' ')
		}
		return "🔓 *Decoded:*\n" + strings.TrimSpace(sb.String()), nil
	}
	var parts []string
	for _, r := range strings.ToUpper(in) {
		if r == ' ' {
			parts = append(parts, "/")
		} else if c, ok := morseEnc[r]; ok {
			parts = append(parts, c)
		}
	}
	return "🔐 *Morse:*\n" + strings.Join(parts, " "), nil
}

func binaryTool(in string, _ *events.Message) (string, error) {
	if strings.Trim(in, "01 ") == "" {
		var out []byte
		for _, b := range strings.Fields(in) {
			n, err := strconv.ParseUint(b, 2, 8)
			if err != nil {
				return "", fmt.Errorf("invalid binary group \"%s\"", b)
			}
			out = append(out, byte(n))
		}
		return "🔓 *Text:*\n" + string(out), nil
	}
	var parts []string
	for _, b := range []byte(in) {
		parts = append(parts, fmt.Sprintf("%08b", b))
	}
	return "🔐 *Binary:*\n" + strings.Join(parts, " "), nil
}

func caesarTool(in string, _ *events.Message) (string, error) {
	f := strings.SplitN(in, " ", 2)
	if len(f) < 2 {
		return "", fmt.Errorf("usage: caesar <shift> <text>")
	}
	shift, err := parseInt(f[0])
	if err != nil {
		return "", err
	}
	s := int(((shift % 26) + 26) % 26)
	var sb strings.Builder
	for _, r := range f[1] {
		switch {
		case r >= 'a' && r <= 'z':
			sb.WriteRune('a' + (r-'a'+rune(s))%26)
		case r >= 'A' && r <= 'Z':
			sb.WriteRune('A' + (r-'A'+rune(s))%26)
		default:
			sb.WriteRune(r)
		}
	}
	return fmt.Sprintf("🔐 *Shift %d:*\n%s", shift, sb.String()), nil
}

func isVowel(r rune) bool { return strings.ContainsRune("aeiouAEIOU", r) }

func pigLatin(in string, _ *events.Message) (string, error) {
	words := strings.Fields(in)
	for i, w := range words {
		runes := []rune(w)
		allLetters := true
		for _, r := range runes {
			if !unicode.IsLetter(r) {
				allLetters = false
			}
		}
		if !allLetters || len(runes) == 0 {
			continue
		}
		if isVowel(runes[0]) {
			words[i] = w + "way"
			continue
		}
		k := 0
		for k < len(runes) && !isVowel(runes[k]) {
			k++
		}
		words[i] = string(runes[k:]) + string(runes[:k]) + "ay"
	}
	return strings.Join(words, " "), nil
}

func reverseRunes(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func wordCount(in string, _ *events.Message) (string, error) {
	words := len(strings.Fields(in))
	chars := len([]rune(in))
	noSpace := len([]rune(strings.Join(strings.Fields(in), "")))
	lines := len(strings.Split(in, "\n"))
	sentences := strings.Count(in, ".") + strings.Count(in, "!") + strings.Count(in, "?")
	return fmt.Sprintf("📝 *Words:* %d\n🔤 *Characters:* %d\n🔡 *No spaces:* %d\n📄 *Lines:* %d\n💬 *Sentences:* %d",
		words, chars, noSpace, lines, sentences), nil
}

func palindrome(in string, _ *events.Message) (string, error) {
	var clean []rune
	for _, r := range strings.ToLower(in) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			clean = append(clean, r)
		}
	}
	c := string(clean)
	if c != "" && c == reverseRunes(c) {
		return "✅ Yes, it's a *palindrome*!", nil
	}
	return "❌ No, it's *not* a palindrome.", nil
}

func mockText(in string, _ *events.Message) (string, error) {
	var sb strings.Builder
	up := false
	for _, r := range in {
		if unicode.IsLetter(r) {
			if up {
				sb.WriteRune(unicode.ToUpper(r))
			} else {
				sb.WriteRune(unicode.ToLower(r))
			}
			up = !up
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String(), nil
}

func aesthetic(in string, _ *events.Message) (string, error) {
	var sb strings.Builder
	for _, r := range in {
		switch {
		case r == ' ':
			sb.WriteRune('\u3000')
		case r > 32 && r < 127:
			sb.WriteRune(r + 0xFEE0)
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String(), nil
}

func vowelCount(in string, _ *events.Message) (string, error) {
	counts := map[rune]int{}
	total, cons := 0, 0
	for _, r := range strings.ToLower(in) {
		if strings.ContainsRune("aeiou", r) {
			counts[r]++
			total++
		} else if unicode.IsLetter(r) {
			cons++
		}
	}
	return fmt.Sprintf("🔤 *Vowels:* %d (a:%d e:%d i:%d o:%d u:%d)\n🔡 *Consonants:* %d",
		total, counts['a'], counts['e'], counts['i'], counts['o'], counts['u'], cons), nil
}

func leetSpeak(in string, _ *events.Message) (string, error) {
	m := map[rune]rune{'a': '4', 'e': '3', 'i': '1', 'o': '0', 's': '5', 't': '7', 'b': '8', 'g': '9'}
	var sb strings.Builder
	for _, r := range in {
		if x, ok := m[unicode.ToLower(r)]; ok {
			sb.WriteRune(x)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String(), nil
}

func strike(in string, _ *events.Message) (string, error) {
	var sb strings.Builder
	for _, r := range in {
		sb.WriteRune(r)
		if r != ' ' && r != '\n' {
			sb.WriteRune('\u0336')
		}
	}
	return sb.String(), nil
}

func zalgo(in string, _ *events.Message) (string, error) {
	r := []rune(in)
	if len(r) > 120 {
		r = r[:120]
	}
	var sb strings.Builder
	for _, c := range r {
		sb.WriteRune(c)
		if c == ' ' || c == '\n' {
			continue
		}
		for i := 0; i < 1+rand.Intn(4); i++ {
			sb.WriteRune(rune(0x0300 + rand.Intn(0x6F)))
		}
	}
	return sb.String(), nil
}

func wordFreq(in string, _ *events.Message) (string, error) {
	freq := map[string]int{}
	for _, w := range strings.FieldsFunc(strings.ToLower(in), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		freq[w]++
	}
	type kv struct {
		k string
		v int
	}
	var list []kv
	for k, v := range freq {
		list = append(list, kv{k, v})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].v == list[j].v {
			return list[i].k < list[j].k
		}
		return list[i].v > list[j].v
	})
	if len(list) > 10 {
		list = list[:10]
	}
	var sb strings.Builder
	for i, e := range list {
		sb.WriteString(fmt.Sprintf("%d. %s — %d\n", i+1, e.k, e.v))
	}
	return strings.TrimSpace(sb.String()), nil
}

// ── math / number tools ─────────────────────────────────────────────────────

func percentTool(in string, _ *events.Message) (string, error) {
	n, err := parseFloats(in, 2)
	if err != nil {
		return "", err
	}
	x, y := n[0], n[1]
	out := fmt.Sprintf("📊 *%g%% of %g* = %.4g", x, y, x/100*y)
	if y != 0 {
		out += fmt.Sprintf("\n📈 *%g is* %.4g%% *of %g*", x, x/y*100, y)
	}
	return out, nil
}

func tempConv(in string, _ *events.Message) (string, error) {
	f := strings.Fields(in)
	if len(f) < 2 {
		return "", fmt.Errorf("usage: tempconv <value> <C|F|K>")
	}
	val, err := strconv.ParseFloat(f[0], 64)
	if err != nil {
		return "", fmt.Errorf("\"%s\" is not a number", f[0])
	}
	var c float64
	switch strings.ToUpper(f[1]) {
	case "C":
		c = val
	case "F":
		c = (val - 32) * 5 / 9
	case "K":
		c = val - 273.15
	default:
		return "", fmt.Errorf("unit must be C, F or K")
	}
	return fmt.Sprintf("🌡 *%.2f °C*\n🌡 *%.2f °F*\n🌡 *%.2f K*", c, c*9/5+32, c+273.15), nil
}

var romanVals = []struct {
	v int
	s string
}{{1000, "M"}, {900, "CM"}, {500, "D"}, {400, "CD"}, {100, "C"}, {90, "XC"}, {50, "L"}, {40, "XL"}, {10, "X"}, {9, "IX"}, {5, "V"}, {4, "IV"}, {1, "I"}}

func romanTool(in string, _ *events.Message) (string, error) {
	in = strings.ToUpper(strings.TrimSpace(in))
	if n, err := strconv.Atoi(in); err == nil {
		if n < 1 || n > 3999 {
			return "", fmt.Errorf("number must be 1–3999")
		}
		var sb strings.Builder
		for _, r := range romanVals {
			for n >= r.v {
				sb.WriteString(r.s)
				n -= r.v
			}
		}
		return "🏛 " + sb.String(), nil
	}
	total := 0
	rest := in
	for _, r := range romanVals {
		for strings.HasPrefix(rest, r.s) {
			total += r.v
			rest = rest[len(r.s):]
		}
	}
	if rest != "" || total == 0 {
		return "", fmt.Errorf("invalid roman numeral")
	}
	return fmt.Sprintf("🔢 %d", total), nil
}

func isPrime(n int64) bool {
	if n < 2 {
		return false
	}
	if n%2 == 0 {
		return n == 2
	}
	for i := int64(3); i*i <= n; i += 2 {
		if n%i == 0 {
			return false
		}
	}
	return true
}

func primeTool(in string, _ *events.Message) (string, error) {
	n, err := parseInt(in)
	if err != nil {
		return "", err
	}
	if n < 0 || n > 1_000_000_000_000 {
		return "", fmt.Errorf("number must be between 0 and 10^12")
	}
	next := n + 1
	for !isPrime(next) {
		next++
	}
	if isPrime(n) {
		return fmt.Sprintf("✅ *%d is prime*\n➡️ Next prime: %d", n, next), nil
	}
	return fmt.Sprintf("❌ *%d is not prime*\n➡️ Next prime: %d", n, next), nil
}

func factorsTool(in string, _ *events.Message) (string, error) {
	n, err := parseInt(in)
	if err != nil {
		return "", err
	}
	if n < 2 || n > 1_000_000_000_000 {
		return "", fmt.Errorf("number must be between 2 and 10^12")
	}
	var parts []string
	m := n
	for p := int64(2); p*p <= m; p++ {
		cnt := 0
		for m%p == 0 {
			m /= p
			cnt++
		}
		if cnt == 1 {
			parts = append(parts, fmt.Sprint(p))
		} else if cnt > 1 {
			parts = append(parts, fmt.Sprintf("%d^%d", p, cnt))
		}
	}
	if m > 1 {
		parts = append(parts, fmt.Sprint(m))
	}
	return fmt.Sprintf("🔬 *%d* = %s", n, strings.Join(parts, " × ")), nil
}

func fibTool(in string, _ *events.Message) (string, error) {
	n, err := parseInt(in)
	if err != nil {
		return "", err
	}
	if n < 1 || n > 90 {
		return "", fmt.Errorf("number must be 1–90")
	}
	a, b := uint64(0), uint64(1)
	seq := make([]string, 0, n)
	for i := int64(0); i < n; i++ {
		seq = append(seq, fmt.Sprint(a))
		a, b = b, a+b
	}
	return fmt.Sprintf("🌀 %s\n\n*F(%d)* = %s", strings.Join(seq, ", "), n-1, seq[len(seq)-1]), nil
}

func factorialTool(in string, _ *events.Message) (string, error) {
	n, err := parseInt(in)
	if err != nil {
		return "", err
	}
	if n < 0 || n > 200 {
		return "", fmt.Errorf("number must be 0–200")
	}
	r := big.NewInt(1)
	for i := int64(2); i <= n; i++ {
		r.Mul(r, big.NewInt(i))
	}
	s := r.String()
	if len(s) > 400 {
		s = s[:400] + "…"
	}
	return fmt.Sprintf("❗ *%d!* = %s", n, s), nil
}

func gcdOf(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a
	}
	return a
}

func gcdTool(in string, _ *events.Message) (string, error) {
	f := strings.Fields(strings.ReplaceAll(in, ",", " "))
	if len(f) < 2 {
		return "", fmt.Errorf("give at least 2 numbers")
	}
	g, l := int64(0), int64(1)
	for _, s := range f {
		n, err := parseInt(s)
		if err != nil {
			return "", err
		}
		if n == 0 {
			return "", fmt.Errorf("numbers must be non-zero")
		}
		g = gcdOf(g, n)
		l = l / gcdOf(l, n) * n
		if l < 0 {
			l = -l
		}
	}
	return fmt.Sprintf("➗ *GCD:* %d\n✖️ *LCM:* %d", g, l), nil
}

func tipTool(in string, _ *events.Message) (string, error) {
	n, err := parseFloats(in, 2)
	if err != nil {
		return "", err
	}
	people := 1.0
	if f := strings.Fields(in); len(f) >= 3 {
		if p, err := strconv.ParseFloat(f[2], 64); err == nil && p >= 1 {
			people = p
		}
	}
	tip := n[0] * n[1] / 100
	total := n[0] + tip
	return fmt.Sprintf("💵 *Bill:* %.2f\n🪙 *Tip (%g%%):* %.2f\n🧾 *Total:* %.2f\n👥 *Per person (%g):* %.2f",
		n[0], n[1], tip, total, people, total/people), nil
}

func discountTool(in string, _ *events.Message) (string, error) {
	n, err := parseFloats(in, 2)
	if err != nil {
		return "", err
	}
	save := n[0] * n[1] / 100
	return fmt.Sprintf("🏷 *Price:* %.2f\n💸 *Discount (%g%%):* -%.2f\n✅ *Final:* %.2f", n[0], n[1], save, n[0]-save), nil
}

func baseConv(in string, _ *events.Message) (string, error) {
	f := strings.Fields(in)
	if len(f) < 3 {
		return "", fmt.Errorf("usage: baseconv <number> <from> <to>")
	}
	from, e1 := strconv.Atoi(f[1])
	to, e2 := strconv.Atoi(f[2])
	if e1 != nil || e2 != nil || from < 2 || from > 36 || to < 2 || to > 36 {
		return "", fmt.Errorf("bases must be between 2 and 36")
	}
	n, err := strconv.ParseInt(f[0], from, 64)
	if err != nil {
		return "", fmt.Errorf("\"%s\" is not valid in base %d", f[0], from)
	}
	return fmt.Sprintf("🔁 %s (base %d) = *%s* (base %d)", f[0], from, strings.ToUpper(strconv.FormatInt(n, to)), to), nil
}

var smallNums = []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten",
	"eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen"}
var tensNames = []string{"", "", "twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety"}

func below1000(n int64) string {
	var parts []string
	if n >= 100 {
		parts = append(parts, smallNums[n/100]+" hundred")
		n %= 100
	}
	if n >= 20 {
		w := tensNames[n/10]
		if n%10 != 0 {
			w += "-" + smallNums[n%10]
		}
		parts = append(parts, w)
	} else if n > 0 {
		parts = append(parts, smallNums[n])
	}
	return strings.Join(parts, " ")
}

func numToWords(in string, _ *events.Message) (string, error) {
	n, err := parseInt(in)
	if err != nil {
		return "", err
	}
	if n < 0 || n > 999_999_999_999 {
		return "", fmt.Errorf("number must be 0–999,999,999,999")
	}
	if n == 0 {
		return "zero", nil
	}
	units := []string{"", " thousand", " million", " billion"}
	var parts []string
	for i := 0; n > 0; i++ {
		if chunk := n % 1000; chunk > 0 {
			parts = append([]string{below1000(chunk) + units[i]}, parts...)
		}
		n /= 1000
	}
	return "🔢 " + strings.Join(parts, " "), nil
}

// ── fun / random ────────────────────────────────────────────────────────────

func randomColor(in string, _ *events.Message) (string, error) {
	r, g, b := rand.Intn(256), rand.Intn(256), rand.Intn(256)
	return fmt.Sprintf("🎨 *HEX:* #%02X%02X%02X\n🎨 *RGB:* rgb(%d, %d, %d)", r, g, b, r, g, b), nil
}

func luckyNumber(in string, _ *events.Message) (string, error) {
	return fmt.Sprintf("🍀 *Lucky number:* %d\n🎯 *Also try:* %d, %d, %d",
		1+rand.Intn(99), 1+rand.Intn(99), 1+rand.Intn(99), 1+rand.Intn(99)), nil
}

func coinFlip(in string, _ *events.Message) (string, error) {
	if rand.Intn(2) == 0 {
		return "🪙 *HEADS*", nil
	}
	return "🪙 *TAILS*", nil
}

func rouletteTool(in string, _ *events.Message) (string, error) {
	n := rand.Intn(37)
	color := "🟢 green"
	reds := map[int]bool{1: true, 3: true, 5: true, 7: true, 9: true, 12: true, 14: true, 16: true, 18: true, 19: true,
		21: true, 23: true, 25: true, 27: true, 30: true, 32: true, 34: true, 36: true}
	if n != 0 {
		if reds[n] {
			color = "🔴 red"
		} else {
			color = "⚫ black"
		}
	}
	return fmt.Sprintf("🎡 The ball lands on *%d* (%s)", n, color), nil
}

func slotsTool(in string, _ *events.Message) (string, error) {
	sym := []string{"🍒", "🍋", "🍇", "🔔", "💎", "7️⃣"}
	a, b, c := pick(sym), pick(sym), pick(sym)
	res := "😢 No luck, try again!"
	switch {
	case a == b && b == c:
		res = "🎉 *JACKPOT!*"
	case a == b || b == c || a == c:
		res = "✨ Two of a kind — nice!"
	}
	return fmt.Sprintf("🎰 [ %s | %s | %s ]\n%s", a, b, c, res), nil
}

func meterTool(label string) func(string, *events.Message) (string, error) {
	return func(in string, v *events.Message) (string, error) {
		name := strings.TrimSpace(in)
		if name == "" {
			name = v.Info.PushName
		}
		if name == "" {
			name = "You"
		}
		p := rand.Intn(101)
		filled := p / 10
		bar := strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
		return fmt.Sprintf("🧪 *%s* is *%d%%* %s\n[%s]", name, p, label, bar), nil
	}
}

var spiritAnimals = []string{"🐺 Wolf", "🦅 Eagle", "🐯 Tiger", "🐉 Dragon", "🦊 Fox", "🐻 Bear", "🦉 Owl", "🐬 Dolphin", "🦁 Lion", "🐍 Snake", "🐆 Panther", "🦈 Shark"}

func spiritAnimal(in string, v *events.Message) (string, error) {
	name := strings.ToLower(strings.TrimSpace(in))
	if name == "" {
		name = strings.ToLower(v.Info.PushName)
	}
	h := 0
	for _, r := range name {
		h = (h*31 + int(r)) % 100003
	}
	return fmt.Sprintf("🔮 Your spirit animal is the *%s*", spiritAnimals[h%len(spiritAnimals)]), nil
}

func zodiacTool(in string, _ *events.Message) (string, error) {
	f := strings.FieldsFunc(in, func(r rune) bool { return r == '-' || r == '/' || r == ' ' || r == '.' })
	if len(f) < 2 {
		return "", fmt.Errorf("usage: zodiac <day> <month>  (e.g. zodiac 14 8)")
	}
	d, e1 := strconv.Atoi(f[0])
	m, e2 := strconv.Atoi(f[1])
	if e1 != nil || e2 != nil || d < 1 || d > 31 || m < 1 || m > 12 {
		return "", fmt.Errorf("invalid date")
	}
	signs := []struct {
		m, d int
		n    string
	}{{1, 20, "♒ Aquarius"}, {2, 19, "♓ Pisces"}, {3, 21, "♈ Aries"}, {4, 20, "♉ Taurus"}, {5, 21, "♊ Gemini"},
		{6, 21, "♋ Cancer"}, {7, 23, "♌ Leo"}, {8, 23, "♍ Virgo"}, {9, 23, "♎ Libra"}, {10, 23, "♏ Scorpio"},
		{11, 22, "♐ Sagittarius"}, {12, 22, "♑ Capricorn"}}
	sign := "♑ Capricorn" // 1 Jan – 19 Jan
	for _, sg := range signs {
		if m > sg.m || (m == sg.m && d >= sg.d) {
			sign = sg.n
		}
	}
	return "🔮 Your sign: *" + sign + "*", nil
}

func ownerTool(in string, _ *events.Message) (string, error) {
	return "👑 *Owner:* " + OwnerURL + "\n📢 *Updates:* " + UpdateURL + "\n💬 *Support:* " + ChatURL, nil
}

func uptimeTool(in string, _ *events.Message) (string, error) {
	return fmt.Sprintf("⏲ *Uptime:* %s\n🕒 *Started:* %s", FormatUptime(time.Since(StartTime)), StartTime.UTC().Format("02 Jan 2006 15:04 UTC")), nil
}

func init() {
	tools := []tool{
		{"reverse", "reverse", "reverse <text>", func(in string, _ *events.Message) (string, error) { return reverseRunes(in), nil }},
		{"wordcount", "word count", "wordcount <text>", wordCount},
		{"caesar", "caesar cipher", "caesar <shift> <text>", caesarTool},
		{"morse", "morse code", "morse <text|morse>", morseTool},
		{"binary", "binary", "binary <text|bits>", binaryTool},
		{"piglatin", "pig latin", "piglatin <text>", pigLatin},
		{"palindrome", "palindrome", "palindrome <text>", palindrome},
		{"mock", "mock text", "mock <text>", mockText},
		{"aesthetic", "aesthetic", "aesthetic <text>", aesthetic},
		{"vowelcount", "vowel count", "vowelcount <text>", vowelCount},
		{"leetspeak", "leetspeak", "leetspeak <text>", leetSpeak},
		{"smallcaps", "small caps", "smallcaps <text>", func(in string, _ *events.Message) (string, error) { return smallCaps(in), nil }},
		{"strikethrough", "strikethrough", "strikethrough <text>", strike},
		{"zalgo", "zalgo", "zalgo <text>", zalgo},
		{"wordfreq", "word frequency", "wordfreq <text>", wordFreq},
		{"percent", "percent", "percent <x> <y>", percentTool},
		{"tempconv", "temperature", "tempconv <value> <C|F|K>", tempConv},
		{"roman", "roman numerals", "roman <number|numeral>", romanTool},
		{"prime", "prime check", "prime <number>", primeTool},
		{"factors", "prime factors", "factors <number>", factorsTool},
		{"fibonacci", "fibonacci", "fibonacci <n>", fibTool},
		{"factorial", "factorial", "factorial <n>", factorialTool},
		{"gcd", "gcd / lcm", "gcd <a> <b> [c…]", gcdTool},
		{"tip", "tip calculator", "tip <bill> <percent> [people]", tipTool},
		{"discount", "discount", "discount <price> <percent>", discountTool},
		{"baseconv", "base convert", "baseconv <number> <from> <to>", baseConv},
		{"numtowords", "number to words", "numtowords <number>", numToWords},
		{"randomcolor", "random color", "", randomColor},
		{"luckynumber", "lucky number", "", luckyNumber},
		{"coinflip", "coin flip", "", coinFlip},
		{"roulette", "roulette", "", rouletteTool},
		{"slots", "slot machine", "", slotsTool},
		{"simp", "simp meter", "", meterTool("simp")},
		{"stupid", "stupid meter", "", meterTool("stupid")},
		{"spiritanimal", "spirit animal", "", spiritAnimal},
		{"zodiac", "zodiac", "zodiac <day> <month>", zodiacTool},
		{"owner", "owner", "", ownerTool},
		{"uptime", "uptime", "", uptimeTool},
	}

	cmds := make([]CommandInfo, 0, len(tools))
	for _, t := range tools {
		cmds = append(cmds, CommandInfo{Pattern: t.name, Handler: t.handler(), Sudo: true})
	}
	Register(ModuleInfo{
		Name:        "PBXTools",
		Description: "PBX Elite offline toolkit (text, math, number theory, fun)",
		Commands:    cmds,
	})
}
