package modules

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

// safeCalc evaluates basic arithmetic: +, -, *, /, ^, %
// We implement a simple expression evaluator (no exec, safe in Go)
func safeCalc(expr string) (string, error) {
	expr = strings.ReplaceAll(expr, " ", "")
	expr = strings.ReplaceAll(expr, "^", "**")
	result, err := evalExpr(expr)
	if err != nil {
		return "", err
	}
	// format nicely
	if result == math.Trunc(result) {
		return fmt.Sprintf("%.0f", result), nil
	}
	return fmt.Sprintf("%g", result), nil
}

// evalExpr — simple recursive descent parser
func evalExpr(s string) (float64, error) {
	p := &parser{input: s, pos: 0}
	v, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	if p.pos != len(p.input) {
		return 0, fmt.Errorf("unexpected character at position %d", p.pos)
	}
	return v, nil
}

type parser struct {
	input string
	pos   int
}

func (p *parser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

func (p *parser) consume() byte {
	b := p.peek()
	p.pos++
	return b
}

func (p *parser) parseExpr() (float64, error) {
	return p.parseAddSub()
}

func (p *parser) parseAddSub() (float64, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return 0, err
	}
	for p.peek() == '+' || p.peek() == '-' {
		op := p.consume()
		right, err := p.parseMulDiv()
		if err != nil {
			return 0, err
		}
		if op == '+' {
			left += right
		} else {
			left -= right
		}
	}
	return left, nil
}

func (p *parser) parseMulDiv() (float64, error) {
	left, err := p.parsePow()
	if err != nil {
		return 0, err
	}
	for p.peek() == '*' || p.peek() == '/' || p.peek() == '%' {
		op := p.consume()
		right, err := p.parsePow()
		if err != nil {
			return 0, err
		}
		switch op {
		case '*':
			left *= right
		case '/':
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			left /= right
		case '%':
			left = math.Mod(left, right)
		}
	}
	return left, nil
}

func (p *parser) parsePow() (float64, error) {
	base, err := p.parseUnary()
	if err != nil {
		return 0, err
	}
	if p.pos+1 < len(p.input) && p.input[p.pos] == '*' && p.input[p.pos+1] == '*' {
		p.pos += 2
		exp, err := p.parseUnary()
		if err != nil {
			return 0, err
		}
		return math.Pow(base, exp), nil
	}
	return base, nil
}

func (p *parser) parseUnary() (float64, error) {
	if p.peek() == '-' {
		p.consume()
		v, err := p.parseAtom()
		return -v, err
	}
	if p.peek() == '+' {
		p.consume()
	}
	return p.parseAtom()
}

func (p *parser) parseAtom() (float64, error) {
	if p.peek() == '(' {
		p.consume()
		v, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		if p.peek() != ')' {
			return 0, fmt.Errorf("missing closing parenthesis")
		}
		p.consume()
		return v, nil
	}
	start := p.pos
	for p.peek() >= '0' && p.peek() <= '9' || p.peek() == '.' {
		p.pos++
	}
	if p.pos == start {
		return 0, fmt.Errorf("unexpected token at pos %d", p.pos)
	}
	return strconv.ParseFloat(p.input[start:p.pos], 64)
}

func calcHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		return ReplyErr(v, "Usage: `.calc <expression>`\nExample: `.calc 5*9+2`")
	}
	expr := strings.Join(args, "")
	result, err := safeCalc(expr)
	if err != nil {
		return ReplyErr(v, fmt.Sprintf("❌ Error: %v", err))
	}
	return ReplyErr(v, fmt.Sprintf("🧮 *Calculator*\n\nExpression: `%s`\nResult: `%s`", expr, result))
}

func mathHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		return ReplyErr(v, "Usage: `.math <operation> <value>`\nOperations: sin, cos, tan, square, cube, sqroot, factorial, abs, log")
	}
	op := strings.ToLower(args[0])
	val, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return ReplyErr(v, "❌ Invalid number.")
	}

	var result float64
	var label string
	switch op {
	case "sin":
		result = math.Sin(val * math.Pi / 180)
		label = fmt.Sprintf("sin(%g°)", val)
	case "cos":
		result = math.Cos(val * math.Pi / 180)
		label = fmt.Sprintf("cos(%g°)", val)
	case "tan":
		result = math.Tan(val * math.Pi / 180)
		label = fmt.Sprintf("tan(%g°)", val)
	case "square":
		result = val * val
		label = fmt.Sprintf("%g²", val)
	case "cube":
		result = val * val * val
		label = fmt.Sprintf("%g³", val)
	case "sqroot":
		if val < 0 {
			return ReplyErr(v, "❌ Cannot take square root of a negative number.")
		}
		result = math.Sqrt(val)
		label = fmt.Sprintf("√%g", val)
	case "factorial":
		n := int(val)
		if n < 0 || n > 20 {
			return ReplyErr(v, "❌ Factorial only supported for 0–20.")
		}
		result = 1
		for i := 2; i <= n; i++ {
			result *= float64(i)
		}
		label = fmt.Sprintf("%d!", n)
	case "abs":
		result = math.Abs(val)
		label = fmt.Sprintf("|%g|", val)
	case "log":
		if val <= 0 {
			return ReplyErr(v, "❌ Log of non-positive number is undefined.")
		}
		result = math.Log10(val)
		label = fmt.Sprintf("log₁₀(%g)", val)
	default:
		return ReplyErr(v, "❌ Unknown operation. Supported: sin, cos, tan, square, cube, sqroot, factorial, abs, log")
	}
	return ReplyErr(v, fmt.Sprintf("🔢 *Math*\n\n%s = `%g`", label, result))
}

func init() {
	Register(ModuleInfo{
		Name:        "Calculator",
		Description: "Evaluate math expressions and perform math operations",
		Commands: []CommandInfo{
			{Pattern: "calculate", Handler: calcHandler, Sudo: true},
			{Pattern: "math", Handler: mathHandler, Sudo: true},
		},
	})
}
