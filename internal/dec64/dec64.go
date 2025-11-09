package dec64

// ⚠️ WARNING ⚠️
// This is an experimental dec64 inspired decimal number implementation with significant limitations:
// - it does not follow the spec, it may never follow the spec and will change over time!
// - Limited precision (56-bit coefficients)
// - Not thoroughly tested for all edge cases
// - May have rounding errors in certain operations
// - No guarantees of numerical stability

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type Dec64 int64

const (
	// MAX_COEFF max signed int 7FFFFFFFFFFFFF FF
	MAX_COEFF = int64(0x7FFFFFFFFFFFFF) // signed 56-bit max
	MIN_COEFF = -MAX_COEFF

	ZERO = Dec64(0)
	ONE  = Dec64((1 << 8) | int64(uint8(0)))
	TEN  = Dec64((1 << 8) | int64(uint8(1)))
	NAN  = Dec64(0x0000000000000080)
	MAX  = Dec64(MAX_COEFF<<8 | int64(uint8(0x7F))) // 36028797018963967×10^127
	MIN  = Dec64(MIN_COEFF<<8 | int64(uint8(0x81))) // -36028797018963967×10^-127
)

// RoundingMode defines the rounding rule for division operations
type RoundingMode int

const (
	RoundHalfUp   RoundingMode = iota // Default rounding mode
	RoundHalfEven                     // Banker's rounding
	RoundDown                         // Always toward zero
	RoundUp                           // Always away from zero
)

// New creates a Dec64 from coefficient and exponent
func New(coef int64, exp int8) Dec64 {
	return Dec64((coef << 8) | int64(uint8(exp)))
}

func FromInt(coef int) Dec64 {
	return FromInt64(int64(coef))
}

func FromUint(coef uint64) Dec64 {
	return FromInt64(int64(coef))
}

func FromInt64(coef int64) Dec64 {
	if coef == 0 {
		return ZERO
	}

	exp := int8(0)

	for coef%10 == 0 && coef > MAX_COEFF && exp < 127 {
		coef /= 10
		exp++
	}

	return normalizeTowardZero(coef, exp)
}

func FromString(s string) (Dec64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return ZERO, errors.New("empty string")
	}

	// Handle sign
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	} else if s[0] == '+' {
		s = s[1:]
	}

	// Handle scientific notation
	var exp int64
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		basePart := s[:i]
		expPart := s[i+1:]
		s = basePart

		e, err := strconv.ParseInt(expPart, 10, 64)
		if err != nil {
			return ZERO, fmt.Errorf("invalid exponent: %v", err)
		}
		if e < -128 || e > 127 {
			return ZERO, fmt.Errorf("exponent %d outside int8 range", e)
		}

		exp = e
	}

	// Handle decimal
	intPart := s
	fracLen := 0
	if dot := strings.Index(s, "."); dot >= 0 {
		intPart = strings.ReplaceAll(s, ".", "")
		fracLen = len(s) - dot - 1
	}

	// Parse coef
	coef, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return ZERO, fmt.Errorf("invalid number: %v", err)
	}

	if neg {
		coef = -coef
	}

	return New(coef, -int8(fracLen)+int8(exp)).Normalize(), nil
}

// Coefficient extracts the integer part
func (a Dec64) Coefficient() int64 {
	return int64(a) >> 8
}

// Exponent extracts the decimal exponent (base-10)
func (a Dec64) Exponent() int8 {
	return int8(a & 0xFF)
}

func (a Dec64) ToInt64() int64 {
	return a.Coefficient() * int64(math.Pow10(int(a.Exponent())))
}

func (a Dec64) ToInt() int {
	return int(a.ToInt64())
}

// ToFloat64 converts Dec64 to float64.
func (a Dec64) ToFloat64() float64 {
	return float64(a.Coefficient()) * math.Pow10(int(a.Exponent()))
}

func (a Dec64) Add(b Dec64) Dec64 {
	ca, cb, e := normalizePair(a, b)
	return normalizeTowardZero(ca+cb, e)
}

func (a Dec64) Sub(b Dec64) Dec64 {
	return a.Add(b.Neg())
}

func (a Dec64) Neg() Dec64 {
	return New(-a.Coefficient(), a.Exponent())
}

func (a Dec64) Mul(b Dec64) Dec64 {
	ea, eb := a.Exponent(), b.Exponent()
	ca, cb := a.Coefficient(), b.Coefficient()
	return normalizeTowardZero(ca*cb, ea+eb)
}

func (a Dec64) Div(b Dec64, precision int, rounding RoundingMode) Dec64 {
	ea, eb := a.Exponent(), b.Exponent()
	ca, cb := a.Coefficient(), b.Coefficient()

	if cb == 0 {
		return NAN // Prevent division by zero
	}

	// Scale to achieve the desired precision
	scale := pow10(int64(precision))
	ca *= scale

	quotient := ca / cb
	remainder := ca % cb

	// Apply rounding based on the specified rounding mode
	if remainder != 0 {
		switch rounding {
		case RoundHalfUp:
			// Round up if the remainder * 2 >= cb
			if abs64(remainder)*2 >= abs64(cb) {
				if (ca > 0) == (cb > 0) { // Same sign
					quotient++
				} else {
					quotient--
				}
			}
		case RoundHalfEven:
			// Banker's rounding: round to the nearest even number
			if abs64(remainder)*2 > abs64(cb) || (abs64(remainder)*2 == abs64(cb) && quotient%2 != 0) {
				if (ca > 0) == (cb > 0) {
					quotient++
				} else {
					quotient--
				}
			}
		case RoundDown:
			// Truncate toward zero, do nothing
		case RoundUp:
			// Always round away from zero
			if (ca > 0) == (cb > 0) {
				quotient++
			} else {
				quotient--
			}
		}
	}

	// Adjust the exponent to account for precision scaling
	return normalizeTowardZero(quotient, ea-eb-int8(precision))
}

func (a Dec64) Mod(b Dec64) Dec64 {
	if b.IsZero() {
		return NAN
	}

	ca, cb, e := normalizePair(a, b)
	return normalizeTowardZero(ca%cb, e)
}

func (a Dec64) Cmp(b Dec64) int {
	ca, cb, _ := normalizePair(a, b)

	if ca < cb {
		return -1
	} else if ca > cb {
		return 1
	}
	return 0
}

func (a Dec64) Eq(b Dec64) bool {
	return a.Cmp(b) == 0
}

func (a Dec64) Neq(b Dec64) bool {
	return a.Cmp(b) != 0
}

func (a Dec64) Lt(b Dec64) bool {
	return a.Cmp(b) == -1
}

func (a Dec64) Lte(b Dec64) bool {
	return a.Cmp(b) <= 0
}

func (a Dec64) Gt(b Dec64) bool {
	return a.Cmp(b) == 1
}

func (a Dec64) Gte(b Dec64) bool {
	return a.Cmp(b) >= 0
}

func (a Dec64) Abs() Dec64 {
	coef := a.Coefficient()
	if coef < 0 {
		coef = -coef
	}
	return New(coef, a.Exponent())
}

func (a Dec64) IsZero() bool {
	return a.Coefficient() == 0 && !a.IsNaN()
}

func (a Dec64) IsNaN() bool {
	return a.Exponent() == int8(-128)
}

func (a Dec64) Normalize() Dec64 {
	return normalizeTowardZero(a.Coefficient(), a.Exponent())
}

func (a Dec64) String() string {
	if a.IsNaN() {
		return "NaN"
	} else if a.IsZero() {
		return "0"
	}

	neg := a.Coefficient() < 0
	mag := abs64(a.Coefficient())

	// Convert to digits
	digits := []byte(strconv.FormatInt(mag, 10))
	exp := a.Exponent()

	var result string
	switch {
	case exp >= 16 || (exp < 0 && -exp >= int8(len(digits))+16):
		// Use scientific notation for large exponents
		point := 1
		result = string(digits[:point]) + "." + string(digits[point:]) + "e" + strconv.FormatInt(int64(exp)+int64(len(digits))-1, 10)
	case exp >= 0:
		// Append zeroes
		result = string(digits) + strings.Repeat("0", int(exp))
	case -exp < int8(len(digits)):
		// Insert decimal point
		point := len(digits) + int(exp)
		result = string(digits[:point]) + "." + string(digits[point:])
	default:
		// Need leading zeroes
		result = "0." + strings.Repeat("0", int(-exp)-len(digits)) + string(digits)
	}

	if neg {
		result = "-" + result
	}
	return result
}

func (a Dec64) StringRaw() string {
	if a.IsNaN() {
		return "NaN"
	} else if a.IsZero() {
		return "0"
	} else if a.Exponent() == 0 {
		return fmt.Sprintf("%d", a.Coefficient())
	} else {
		return fmt.Sprintf("%d×10^%d", a.Coefficient(), a.Exponent())
	}
}

func Max(values ...Dec64) Dec64 {
	if len(values) == 0 {
		return NAN
	}
	max := values[0]
	for _, v := range values[1:] {
		if v.IsNaN() {
			return NAN
		}
		if v.Gt(max) {
			max = v
		}
	}
	return max
}

func Min(values ...Dec64) Dec64 {
	if len(values) == 0 {
		return NAN
	}
	min := values[0]
	for _, v := range values[1:] {
		if v.IsNaN() {
			return NAN
		}
		if v.Lt(min) {
			min = v
		}
	}
	return min
}

func normalizeTowardZero(coef int64, exp int8) Dec64 {
	if coef == 0 {
		return ZERO
	}

	const maxDigits = 16
	if abs64(coef) >= pow10(maxDigits) {
		for coef%10 == 0 {
			coef /= 10
			exp++
		}
	}

	if exp < 0 {
		for exp < 0 && coef%10 == 0 {
			coef /= 10
			exp++
		}
	} else if exp > 0 && exp < 16 {
		for exp > 0 && coef < MAX_COEFF/10 {
			coef *= 10
			exp--
		}
	}

	return New(coef, exp)
}

func normalizePair(a, b Dec64) (int64, int64, int8) {
	ea, eb := a.Exponent(), b.Exponent()
	ca, cb := a.Coefficient(), b.Coefficient()

	if ea > eb {
		shift := int(ea - eb)
		for i := 0; i < shift; i++ {
			ca *= 10
		}
		return ca, cb, eb
	} else if eb > ea {
		shift := int(eb - ea)
		for i := 0; i < shift; i++ {
			cb *= 10
		}
		return ca, cb, ea
	}
	return ca, cb, ea
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

// Bitwise operations section.
// Policy: only allowed for exact integers (exponent == 0). Otherwise, return NaN.
// For shifts, the shift count must be exponent==0 and non-negative.

// And returns bitwise AND of two integer Dec64 values.
func (a Dec64) And(b Dec64) Dec64 {
	if a.IsNaN() || b.IsNaN() || a.Exponent() != 0 || b.Exponent() != 0 {
		return NAN
	}
	c := a.Coefficient() & b.Coefficient()
	// Ensure result fits signed 56-bit coefficient domain if you want strictness.
	if c > MAX_COEFF || c < MIN_COEFF {
		return NAN
	}
	return New(c, 0)
}

// Or returns bitwise OR of two integer Dec64 values.
func (a Dec64) Or(b Dec64) Dec64 {
	if a.IsNaN() || b.IsNaN() || a.Exponent() != 0 || b.Exponent() != 0 {
		return NAN
	}
	c := a.Coefficient() | b.Coefficient()
	if c > MAX_COEFF || c < MIN_COEFF {
		return NAN
	}
	return New(c, 0)
}

// Xor returns bitwise XOR of two integer Dec64 values.
func (a Dec64) Xor(b Dec64) Dec64 {
	if a.IsNaN() || b.IsNaN() || a.Exponent() != 0 || b.Exponent() != 0 {
		return NAN
	}
	c := a.Coefficient() ^ b.Coefficient()
	if c > MAX_COEFF || c < MIN_COEFF {
		return NAN
	}
	return New(c, 0)
}

// Not returns bitwise NOT of an integer Dec64 value.
func (a Dec64) Not() Dec64 {
	if a.IsNaN() || a.Exponent() != 0 {
		return NAN
	}
	c := ^a.Coefficient()
	if c > MAX_COEFF || c < MIN_COEFF {
		return NAN
	}
	return New(c, 0)
}

// ShiftLeft returns a << n for integer Dec64 a and integer, non-negative shift n.
func (a Dec64) ShiftLeft(n Dec64) Dec64 {
	if a.IsNaN() || n.IsNaN() || a.Exponent() != 0 || n.Exponent() != 0 {
		return NAN
	}
	shift := n.Coefficient()
	if shift < 0 || shift > 62 { // conservative: avoid undefined or excessive shifts
		return NAN
	}
	c := a.Coefficient()
	shifted := c << uint(shift)
	// Check we remain within signed 56-bit coefficient domain
	if shifted > MAX_COEFF || shifted < MIN_COEFF {
		return NAN
	}
	return New(shifted, 0)
}

// ShiftRight returns arithmetic right shift a >> n for integer Dec64 a and integer, non-negative shift n.
func (a Dec64) ShiftRight(n Dec64) Dec64 {
	if a.IsNaN() || n.IsNaN() || a.Exponent() != 0 || n.Exponent() != 0 {
		return NAN
	}
	shift := n.Coefficient()
	if shift < 0 || shift > 62 {
		return NAN
	}
	c := a.Coefficient()
	shifted := c >> uint(shift)
	// Always fits signed 56-bit because shifting right reduces magnitude
	if shifted > MAX_COEFF || shifted < MIN_COEFF {
		return NAN
	}
	return New(shifted, 0)
}
