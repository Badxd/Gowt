package modules

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

// .fact <n>     — factorial
// .fib <n>      — nth fibonacci number
// .prime <n>    — check if prime / list primes up to n
// .sqrt <n>     — square root
// .pow <base> <exp> — power


func fibHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .fib <n>\nExample: .fib 10")
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n < 0 {
		return ReplyErr(v, "❌ Please provide a non-negative integer")
	}
	if n > 80 {
		return ReplyErr(v, "❌ Max is 80")
	}

	if n == 0 {
		return ReplyErr(v, "🔢 *Fib(0) =* 0")
	}
	if n == 1 {
		return ReplyErr(v, "🔢 *Fib(1) =* 1")
	}

	a, b := uint64(0), uint64(1)
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return ReplyErr(v, fmt.Sprintf("🔢 *Fib(%d) =* %d", n, b))
}

func isPrime(n int64) bool {
	if n < 2 {
		return false
	}
	if n == 2 {
		return true
	}
	if n%2 == 0 {
		return false
	}
	for i := int64(3); i <= int64(math.Sqrt(float64(n))); i += 2 {
		if n%i == 0 {
			return false
		}
	}
	return true
}

func primesUpTo(limit int64) []int64 {
	var result []int64
	for i := int64(2); i <= limit; i++ {
		if isPrime(i) {
			result = append(result, i)
		}
	}
	return result
}

func primeHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .prime <n> — check if prime or list primes up to n\nExamples:\n.prime 17\n.prime list 50")
	}

	if args[0] == "list" {
		if len(args) < 2 {
			return ReplyErr(v, "Usage: .prime list <limit>")
		}
		limit, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil || limit < 2 || limit > 1000 {
			return ReplyErr(v, "❌ Limit must be 2–1000")
		}
		primes := primesUpTo(limit)
		strs := make([]string, len(primes))
		for i, p := range primes {
			strs[i] = strconv.FormatInt(p, 10)
		}
		return ReplyErr(v, fmt.Sprintf("🔢 *Primes up to %d* (%d total):\n%s", limit, len(primes), strings.Join(strs, ", ")))
	}

	n, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return ReplyErr(v, "❌ Please provide a valid integer")
	}
	if isPrime(n) {
		return ReplyErr(v, fmt.Sprintf("✅ *%d* is a *prime* number!", n))
	}
	return ReplyErr(v, fmt.Sprintf("❌ *%d* is *not* a prime number.", n))
}

func sqrtHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: .sqrt <number>")
	}
	n, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return ReplyErr(v, "❌ Invalid number")
	}
	if n < 0 {
		return ReplyErr(v, "❌ Cannot take sqrt of negative number (use complex math)")
	}
	result := math.Sqrt(n)
	return ReplyErr(v, fmt.Sprintf("√%s = *%g*", args[0], result))
}

func powHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		return ReplyErr(v, "Usage: .pow <base> <exponent>\nExample: .pow 2 10")
	}
	base, err1 := strconv.ParseFloat(args[0], 64)
	exp, err2 := strconv.ParseFloat(args[1], 64)
	if err1 != nil || err2 != nil {
		return ReplyErr(v, "❌ Invalid numbers")
	}
	result := math.Pow(base, exp)
	return ReplyErr(v, fmt.Sprintf("🔢 *%g^%g =* %g", base, exp, result))
}

func init() {
	Register(ModuleInfo{
		Name:        "MathTools",
		Description: "Math utilities: factorial, fibonacci, prime, sqrt, power",
		Commands: []CommandInfo{
			{Pattern: "fib", Handler: fibHandler, Sudo: true},
			{Pattern: "prime", Handler: primeHandler, Sudo: true},
			{Pattern: "sqrt", Handler: sqrtHandler, Sudo: true},
			{Pattern: "pow", Handler: powHandler, Sudo: true},
		},
	})
}
