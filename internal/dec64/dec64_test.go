package dec64

import (
	"fmt"
	"testing"
)

func TestAdd(t *testing.T) {
	cases := []struct {
		name     string
		a, b     Dec64
		expected Dec64
	}{
		{"1 + 1", New(1, 0), New(1, 0), New(2, 0)},
		{"10 + 1", New(1, 1), New(1, 0), New(11, 0)},
		{"1 + 10", New(1, 0), New(1, 1), New(11, 0)},
		{"10 + 5", New(1, 1), New(5, 0), New(15, 0)},
		{"5 + 10", New(5, 0), New(1, 1), New(15, 0)},
		{"0 + 0", New(0, 0), New(0, 0), ZERO},
		{"1.2 + 3.4", New(12, -1), New(34, -1), New(46, -1)},
		{"1e100 + 1e99", New(1, 100), New(1, 99), New(11, 99)},
		//{"Addition with normalization", New(10, -1), New(1, 0), New(2, 0)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := c.a.Add(c.b)
			if !result.Eq(c.expected) {
				t.Errorf("expected %s, got %s / %s", c.expected.String(), result.String(), result.StringRaw())
			}
		})
	}
}

func TestSub(t *testing.T) {
	cases := []struct {
		name     string
		a, b     Dec64
		expected Dec64
	}{
		{"1 - 2", New(1, 0), New(2, 0), New(-1, 0)},
		{"1e100 - 1e99", New(1, 100), New(1, 99), New(9, 99)},
		{"Subtraction leading to zero", New(1, 0), New(1, 0), ZERO},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := c.a.Sub(c.b)
			if !result.Eq(c.expected) {
				t.Errorf("expected %s, got %s / %s", c.expected.String(), result.String(), result.StringRaw())
			}
		})
	}
}

func TestMul(t *testing.T) {
	cases := []struct {
		name     string
		a, b     Dec64
		expected Dec64
	}{
		{"2 * 3", New(2, 0), New(3, 0), New(6, 0)},
		{"Multiplication by zero", New(1234, -2), ZERO, ZERO},
		{"Negative multiplied with positive", New(-5, 0), New(2, 0), New(-10, 0)},
		{"Negative multiplied with positive", New(-5, 0), New(2, 0), New(-10, 0)},
		//{"Large multiplication", New(MAX_COEFF, 0), New(2, 0), New(-2, 126)}, // Max * 2 overflow (illustrative)
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := c.a.Mul(c.b)
			if !result.Eq(c.expected) {
				t.Errorf("expected %s, got %s / %s", c.expected.String(), result.String(), result.StringRaw())
			}
		})
	}
}

func TestDiv(t *testing.T) {
	cases := []struct {
		name     string
		a, b     Dec64
		expected Dec64
	}{
		{"6 / 3", New(6, 0), New(3, 0), New(2, 0)},
		{"Division by itself", New(123, -1), New(123, -1), New(1, 0)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := c.a.Div(c.b, 14, RoundHalfUp)
			if !result.Eq(c.expected) {
				t.Errorf("expected %s, got %s", c.expected.String(), result.String())
			}
		})
	}
}

func TestStringFormatParse(t *testing.T) {
	cases := []string{
		"0", "1", "-1", "123.456", "-0.001", "-9.9e-9", "42.0", "1e3",
	}

	for _, s := range cases {
		t.Run("Parse "+s, func(t *testing.T) {
			d, err := FromString(s)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			out := d.String()
			reparsed, err := FromString(out)
			if err != nil {
				t.Fatalf("reparse error: %v", err)
			}
			if d != reparsed {
				t.Errorf("expected %s, got %s", d.String(), reparsed.String())
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	cases := []struct {
		in       Dec64
		expected Dec64
	}{
		{New(100, -2), New(1, 0)},
		{New(10, -1), New(1, 0)},
		{New(123000, -3), New(123, 0)},
		{New(1000, -3), New(1, 0)},
		{New(12345000, -5), New(12345, -2)},
		{New(-100, 2), New(-10000, 0)}, // No trailing zero
	}

	for _, c := range cases {
		t.Run(c.in.String(), func(t *testing.T) {
			result := c.in.Normalize()
			if !result.Eq(c.expected) {
				t.Errorf("expected %s, got %s", c.expected.String(), result.String())
			}
		})
	}
}

func TestParseDec64_EdgeCases(t *testing.T) {
	cases := []struct {
		input    string
		expected Dec64
		hasError bool
	}{
		{"", ZERO, true},        // Empty string
		{"0", ZERO, false},      // Zero
		{"1", New(1, 0), false}, // Simple integer
		{"10", New(10, 0), false},
		{"-10", New(-10, 0), false},
		{"123.45", New(12345, -2), false},         // Decimal number
		{"-123.450", New(-12345, -2), false},      // Negative decimal
		{"0.0001", New(1, -4), false},             // Small decimal
		{"-1e3", New(-1000, 0), false},            // Negative scientific notation
		{"9.9e-10", New(99, -11), false},          // Small exponent
		{"9e999", ZERO, true},                     // Invalid large exponent
		{".1", New(1, -1), false},                 // Leading decimal
		{"42.", New(42, 0), false},                // Trailing decimal
		{"90000003e25", New(90000003, 25), false}, // Trailing decimal
		{"invalid", ZERO, true},                   // Invalid string
	}

	for _, c := range cases {
		t.Run("Parse "+c.input, func(t *testing.T) {
			result, err := FromString(c.input)
			if (err != nil) != c.hasError {
				t.Fatalf("expected error: %v, got: %v / %v", c.hasError, err, result)
			}
			if err == nil && result.Neq(c.expected) {
				t.Errorf("expected %v, got %v, %s", c.expected, result, result.StringRaw())
			}
		})
	}
}

func TestStringFormat_EdgeCases(t *testing.T) {
	cases := []struct {
		input    Dec64
		expected string
	}{
		{ZERO, "0"},
		{New(123, -2), "1.23"},
		{New(-12345, -3), "-12.345"},
		{New(1, 10), "10000000000"},
		{New(-1, -10), "-0.0000000001"},
	}

	for _, c := range cases {
		t.Run(c.input.String(), func(t *testing.T) {
			result := c.input.String()
			if result != c.expected {
				t.Errorf("expected: %s, got: %s", c.expected, result)
			}
		})
	}
}

func TestAbs_EdgeCases(t *testing.T) {
	cases := []struct {
		input    Dec64
		expected Dec64
	}{
		{New(-123, 0), New(123, 0)},
		{ZERO, ZERO},
		{New(-1, -2), New(1, -2)},
	}

	for _, c := range cases {
		t.Run(c.input.String(), func(t *testing.T) {
			result := c.input.Abs()
			if !result.Eq(c.expected) {
				t.Errorf("expected %v, got %v", c.expected, result)
			}
		})
	}
}

func TestCmp_EdgeCases(t *testing.T) {
	cases := []struct {
		a, b     Dec64
		expected int
	}{
		//{New(100, 0), New(10, 1), 0},  // todo these are not normalized
		{New(100, 0), New(100, 0), 0},
		{New(1, 0), New(1, 1), -1},
		{New(1, 1), New(1, 0), 1},
	}

	for _, c := range cases {
		t.Run(c.a.String()+" vs "+c.b.String(), func(t *testing.T) {
			result := c.a.Cmp(c.b)
			if result != c.expected {
				t.Errorf("expected: %d, got: %d", c.expected, result)
			}
		})
	}
}

func TestDiv_ZeroNumerator(t *testing.T) {
	result := ZERO.Div(New(123, 0), 14, RoundHalfUp)
	if result != ZERO {
		t.Errorf("expected zero, got %v", result)
	}
}

func TestDiv_ByZero(t *testing.T) {
	result := New(1, 0).Div(ZERO, 14, RoundHalfUp)
	if NAN != result && result.IsNaN() == false {
		t.Errorf("expected NAN, got %v", result)
	}
}

func TestEq(t *testing.T) {
	cases := []struct {
		a, b     Dec64
		expected bool
	}{
		{New(100, 0), New(100, 0), true},
		//{New(1, 1), New(10, 0), true},
		{New(1, 0), New(1, 1), false},
		{ZERO, ZERO, true},
		{New(-1, 0), New(-1, 0), true},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%v == %v", c.a, c.b), func(t *testing.T) {
			result := c.a.Eq(c.b)
			if result != c.expected {
				t.Errorf("expected %v, got %v", c.expected, result)
			}
		})
	}
}

func TestNeq(t *testing.T) {
	cases := []struct {
		a, b     Dec64
		expected bool
	}{
		{New(100, 0), New(100, 0), false},
		{New(10, 0), New(10, 0), false},
		{New(1, 0), New(1, 1), true},
		{ZERO, New(1, 0), true},
		{New(-1, 0), New(1, 0), true},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%v != %v", c.a, c.b), func(t *testing.T) {
			result := c.a.Neq(c.b)
			if result != c.expected {
				t.Errorf("expected %v, got %v", c.expected, result)
			}
		})
	}
}

func TestLt(t *testing.T) {
	cases := []struct {
		a, b     Dec64
		expected bool
	}{
		{New(1, 0), New(2, 0), true},
		{New(1, 0), New(1, 1), true},
		{New(2, 0), New(1, 0), false},
		{New(1, 1), New(1, 0), false},
		{ZERO, New(1, 0), true},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%v < %v", c.a, c.b), func(t *testing.T) {
			result := c.a.Lt(c.b)
			if result != c.expected {
				t.Errorf("expected %v, got %v", c.expected, result)
			}
		})
	}
}

func TestLte(t *testing.T) {
	cases := []struct {
		a, b     Dec64
		expected bool
	}{
		{New(1, 0), New(2, 0), true},
		{New(1, 0), New(1, 0), true},
		{New(2, 0), New(1, 0), false},
		{ZERO, ZERO, true},
		{New(1, 1), New(10, 0), true},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%v <= %v", c.a, c.b), func(t *testing.T) {
			result := c.a.Lte(c.b)
			if result != c.expected {
				t.Errorf("expected %v, got %v", c.expected, result)
			}
		})
	}
}

func TestGt(t *testing.T) {
	cases := []struct {
		a, b     Dec64
		expected bool
	}{
		{New(2, 0), New(1, 0), true},
		{New(1, 1), New(1, 0), true},
		{New(1, 0), New(2, 0), false},
		{New(1, 0), New(1, 1), false},
		{New(1, 0), ZERO, true},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%v > %v", c.a, c.b), func(t *testing.T) {
			result := c.a.Gt(c.b)
			if result != c.expected {
				t.Errorf("expected %v, got %v", c.expected, result)
			}
		})
	}
}

func TestGte(t *testing.T) {
	cases := []struct {
		a, b     Dec64
		expected bool
	}{
		{New(2, 0), New(1, 0), true},
		{New(1, 0), New(1, 0), true},
		{New(1, 0), New(2, 0), false},
		{ZERO, ZERO, true},
		{New(10, 0), New(1, 1), true},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%v >= %v", c.a, c.b), func(t *testing.T) {
			result := c.a.Gte(c.b)
			if result != c.expected {
				t.Errorf("expected %v, got %v", c.expected, result)
			}
		})
	}
}

func TestStringRaw(t *testing.T) {
	cases := []struct {
		name     string
		input    Dec64
		expected string
	}{
		{"Zero", ZERO, "0"},
		{"NaN", NAN, "NaN"},
		{"Positive whole number", New(123, 0), "123"},
		{"Negative whole number", New(-456, 0), "-456"},
		{"Positive with exponent", New(5, 3), "5×10^3"},
		{"Negative with exponent", New(-7, -2), "-7×10^-2"},
		{"Large exponent", New(12, 10), "12×10^10"},
		{"Small negative exponent", New(34, -5), "34×10^-5"},
		{"Positive coefficient with zero exponent", New(42, 0), "42"},
		{"Negative coefficient with zero exponent", New(-42, 0), "-42"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := c.input.StringRaw()
			if result != c.expected {
				t.Errorf("expected: %s, got: %s", c.expected, result)
			}
		})
	}
}

func TestMax(t *testing.T) {
	// Test with an empty slice
	result := Max()
	if result != NAN {
		t.Errorf("Max() with empty slice expected: NAN, got: %v", result)
	}

	// Test with a single value
	result = Max(ONE)
	if result != ONE {
		t.Errorf("Max(ONE) expected: ONE, got: %v", result)
	}

	// Test with multiple values
	result = Max(ZERO, ONE, TEN)
	if result != TEN {
		t.Errorf("Max(ZERO, ONE, TEN) expected: TEN, got: %v", result)
	}

	// Test with negative values
	result = Max(ZERO, New(-5, 0))
	if result != ZERO {
		t.Errorf("Max(ZERO, New(-5, 0)) expected: ZERO, got: %v", result)
	}

	// Test with equal values
	result = Max(ONE, ONE)
	if result != ONE {
		t.Errorf("Max(ONE, ONE) expected: ONE, got: %v", result)
	}

	// Test combining NAN
	result = Max(NAN, ONE, TEN)
	if result != TEN {
		t.Errorf("Max(NAN, ONE, TEN) expected: TEN, got: %v", result)
	}
}

func TestMin(t *testing.T) {
	// Test with an empty slice
	result := Min()
	if result != NAN {
		t.Errorf("Min() with empty slice expected: NAN, got: %v", result)
	}

	// Test with a single value
	result = Min(ONE)
	if result != ONE {
		t.Errorf("Min(ONE) expected: ONE, got: %v", result)
	}

	// Test with multiple values
	result = Min(ZERO, ONE, TEN)
	if result != ZERO {
		t.Errorf("Min(ZERO, ONE, TEN) expected: ZERO, got: %v", result)
	}

	// Test with negative values
	result = Min(New(-5, 0), ZERO)
	if result != New(-5, 0) {
		t.Errorf("Min(New(-5, 0), ZERO) expected: New(-5, 0), got: %v", result)
	}

	// Test with equal values
	result = Min(ONE, ONE)
	if result != ONE {
		t.Errorf("Min(ONE, ONE) expected: ONE, got: %v", result)
	}

	// Test combining NAN
	result = Min(NAN, ONE, ZERO)
	if result != NAN {
		t.Errorf("Min(NAN, ONE, ZERO) expected: NAN, got: %v", result)
	}
}
