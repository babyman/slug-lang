package foreign

import (
	"fmt"
	"strconv"
	"strings"

	"slug/internal/dec64"
	"slug/internal/object"
)

const defaultFloatPrecision = 6

type formatSpec struct {
	align     rune
	width     int
	precision *int
	verb      string
	grouping  bool
	percent   bool
}

type placeholder struct {
	index  int
	isAuto bool
	spec   formatSpec
}

func formatWithSpec(format string, args []object.Object) (string, error) {
	var out strings.Builder
	autoIndex := 0

	for i := 0; i < len(format); {
		ch := format[i]
		switch ch {
		case '\\':
			if i+1 >= len(format) {
				return "", fmt.Errorf("dangling escape at end of format string")
			}
			next := format[i+1]
			if next != '{' && next != '}' {
				return "", fmt.Errorf("invalid escape sequence: \\%c", next)
			}
			out.WriteByte(next)
			i += 2
		case '{':
			end := strings.IndexByte(format[i+1:], '}')
			if end == -1 {
				return "", fmt.Errorf("unmatched '{' in format string")
			}
			end += i + 1
			content := format[i+1 : end]
			ph, err := parsePlaceholder(content)
			if err != nil {
				return "", fmt.Errorf("invalid placeholder {%s}: %w", content, err)
			}

			argIndex := ph.index
			if ph.isAuto {
				argIndex = autoIndex
				autoIndex++
			}
			if argIndex < 0 || argIndex >= len(args) {
				return "", fmt.Errorf("placeholder {%s} references missing argument %d", content, argIndex)
			}

			rendered, err := renderArgument(args[argIndex], ph.spec)
			if err != nil {
				return "", fmt.Errorf("placeholder {%s} %w", content, err)
			}
			out.WriteString(rendered)
			i = end + 1
		case '}':
			return "", fmt.Errorf("unescaped '}' in format string")
		default:
			out.WriteByte(ch)
			i++
		}
	}

	return out.String(), nil
}

func parsePlaceholder(content string) (placeholder, error) {
	if content == "" {
		return placeholder{isAuto: true}, nil
	}

	ph := placeholder{}
	if content[0] == ':' {
		spec, err := parseFormatSpec(content[1:])
		if err != nil {
			return placeholder{}, err
		}
		ph.isAuto = true
		ph.spec = spec
		return ph, nil
	}

	i := 0
	for i < len(content) && content[i] >= '0' && content[i] <= '9' {
		i++
	}
	if i == 0 {
		return placeholder{}, fmt.Errorf("expected positional index or ':'")
	}

	indexVal, err := strconv.Atoi(content[:i])
	if err != nil {
		return placeholder{}, fmt.Errorf("invalid index")
	}
	ph.index = indexVal
	ph.isAuto = false

	if i == len(content) {
		return ph, nil
	}
	if content[i] != ':' {
		return placeholder{}, fmt.Errorf("unexpected character '%c'", content[i])
	}

	spec, err := parseFormatSpec(content[i+1:])
	if err != nil {
		return placeholder{}, err
	}
	ph.spec = spec
	return ph, nil
}

func parseFormatSpec(spec string) (formatSpec, error) {
	if spec == "" {
		return formatSpec{}, nil
	}

	fs := formatSpec{}
	pos := 0

	if pos < len(spec) {
		switch spec[pos] {
		case '<', '>', '^':
			fs.align = rune(spec[pos])
			pos++
		}
	}

	widthStart := pos
	for pos < len(spec) && spec[pos] >= '0' && spec[pos] <= '9' {
		pos++
	}
	if pos > widthStart {
		widthVal, err := strconv.Atoi(spec[widthStart:pos])
		if err != nil {
			return formatSpec{}, fmt.Errorf("invalid width")
		}
		fs.width = widthVal
	}

	for pos < len(spec) {
		switch spec[pos] {
		case '.':
			if fs.precision != nil {
				return formatSpec{}, fmt.Errorf("duplicate precision")
			}
			pos++
			precStart := pos
			for pos < len(spec) && spec[pos] >= '0' && spec[pos] <= '9' {
				pos++
			}
			if precStart == pos {
				return formatSpec{}, fmt.Errorf("precision missing digits")
			}
			precVal, err := strconv.Atoi(spec[precStart:pos])
			if err != nil {
				return formatSpec{}, fmt.Errorf("invalid precision")
			}
			fs.precision = &precVal
		case 'f', 'd', 's':
			if fs.verb != "" {
				return formatSpec{}, fmt.Errorf("duplicate verb")
			}
			fs.verb = spec[pos : pos+1]
			pos++
		case ',':
			if fs.grouping {
				return formatSpec{}, fmt.Errorf("duplicate grouping")
			}
			fs.grouping = true
			pos++
		case '%':
			if fs.percent {
				return formatSpec{}, fmt.Errorf("duplicate percent flag")
			}
			fs.percent = true
			pos++
		default:
			return formatSpec{}, fmt.Errorf("invalid format specifier '%c'", spec[pos])
		}
	}

	return fs, nil
}

func renderArgument(arg object.Object, spec formatSpec) (string, error) {
	if num, ok := arg.(*object.Number); ok {
		return renderNumber(num.Value, spec)
	}
	if spec.precision != nil || spec.grouping || spec.percent {
		return "", fmt.Errorf("requires numeric value")
	}
	if spec.verb != "" && spec.verb != "s" {
		return "", fmt.Errorf("unsupported verb %q for %s", spec.verb, arg.Type())
	}

	base := arg.Inspect()
	if str, ok := arg.(*object.String); ok {
		base = str.Value
	}

	base = applyWidth(base, spec.width, spec.align)
	return base, nil
}

func renderNumber(val dec64.Dec64, spec formatSpec) (string, error) {
	if spec.verb == "s" {
		return "", fmt.Errorf("unsupported verb %q for number", spec.verb)
	}

	if spec.percent {
		val = val.Mul(dec64.New(100, 0))
	}

	if spec.verb == "d" {
		zero := 0
		spec.precision = &zero
	}

	if spec.precision != nil || spec.verb == "f" || spec.percent {
		precision := defaultFloatPrecision
		if spec.precision != nil {
			precision = *spec.precision
		}
		text := formatFixedNumber(val, precision, spec.grouping)
		if spec.percent {
			text += "%"
		}
		text = applyWidth(text, spec.width, spec.align)
		return text, nil
	}

	base := val.String()
	if spec.grouping {
		base = applyGroupingIfDecimal(base)
	}
	base = applyWidth(base, spec.width, spec.align)
	return base, nil
}

func formatFixedNumber(val dec64.Dec64, precision int, grouping bool) string {
	if val.IsNaN() {
		return "NaN"
	}

	coef := val.Coefficient()
	exp := val.Exponent()

	if precision < 0 {
		precision = 0
	}
	targetExp := int8(-precision)

	if exp < targetExp {
		delta := int64(targetExp - exp)
		if delta > 18 {
			coef = 0
		} else {
			divisor := pow10(delta)
			q := coef / divisor
			r := coef % divisor
			if r != 0 {
				absR := abs64(r)
				if absR*2 > divisor || (absR*2 == divisor && q%2 != 0) {
					if coef > 0 {
						q++
					} else {
						q--
					}
				}
			}
			coef = q
		}
		exp = targetExp
	}

	if coef == 0 {
		exp = targetExp
	}

	neg := coef < 0
	if coef == 0 {
		neg = false
	}
	digits := strconv.FormatInt(abs64(coef), 10)

	var intPart string
	var fracPart string

	if exp >= 0 {
		intPart = digits + strings.Repeat("0", int(exp))
	} else {
		point := len(digits) + int(exp)
		if point > 0 {
			intPart = digits[:point]
			fracPart = digits[point:]
		} else {
			intPart = "0"
			fracPart = strings.Repeat("0", -point) + digits
		}
	}

	if precision > 0 {
		if len(fracPart) < precision {
			fracPart += strings.Repeat("0", precision-len(fracPart))
		} else if len(fracPart) > precision {
			fracPart = fracPart[:precision]
		}
	} else {
		fracPart = ""
	}

	if grouping {
		intPart = addGrouping(intPart)
	}

	result := intPart
	if precision > 0 {
		result = intPart + "." + fracPart
	}

	if neg {
		result = "-" + result
	}

	return result
}

func applyGroupingIfDecimal(text string) string {
	if strings.ContainsAny(text, "eE") {
		return text
	}
	dot := strings.IndexByte(text, '.')
	if dot == -1 {
		return addGrouping(text)
	}
	intPart := text[:dot]
	frac := text[dot+1:]
	return addGrouping(intPart) + "." + frac
}

func addGrouping(text string) string {
	if text == "" {
		return text
	}
	neg := false
	if text[0] == '-' {
		neg = true
		text = text[1:]
	}
	n := len(text)
	if n <= 3 {
		if neg {
			return "-" + text
		}
		return text
	}
	var out strings.Builder
	lead := n % 3
	if lead == 0 {
		lead = 3
	}
	out.WriteString(text[:lead])
	for i := lead; i < n; i += 3 {
		out.WriteByte(',')
		out.WriteString(text[i : i+3])
	}
	if neg {
		return "-" + out.String()
	}
	return out.String()
}

func applyWidth(text string, width int, align rune) string {
	if width <= 0 || len(text) >= width {
		return text
	}
	pad := width - len(text)
	switch align {
	case '<':
		return text + strings.Repeat(" ", pad)
	case '^':
		left := pad / 2
		right := pad - left
		return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
	default:
		return strings.Repeat(" ", pad) + text
	}
}

func formatExcerpt(text string, maxLen int) string {
	if maxLen <= 0 || len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func pow10(n int64) int64 {
	r := int64(1)
	for i := int64(0); i < n; i++ {
		r *= 10
	}
	return r
}
