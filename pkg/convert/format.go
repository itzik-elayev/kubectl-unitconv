package convert

import (
	"fmt"

	"gopkg.in/inf.v0"
)

// MinPrecision and MaxPrecision bound the --precision flag. 18 comfortably
// exceeds any meaningful display precision for a unit-conversion CLI (it
// matches the largest decimal SI exponent, Exa, that this tool itself uses)
// while keeping output readable and preventing pathological formatting.
const (
	MinPrecision = 0
	MaxPrecision = 18
)

// ValidatePrecision rejects a --precision value outside [MinPrecision,
// MaxPrecision]. Negative precision is rejected here rather than passed
// through to formatting, where it could corrupt integer output via
// trailing-zero trimming.
func ValidatePrecision(precision int) error {
	if precision < MinPrecision || precision > MaxPrecision {
		return fmt.Errorf("invalid --precision %d: must be between %d and %d", precision, MinPrecision, MaxPrecision)
	}
	return nil
}

// formatDec renders an inf.Dec already rounded to `precision` decimal
// digits, trimming trailing fractional zeros (and a trailing '.') for a
// clean display value. precision == 0 means the value is a whole number;
// there is no decimal point to trim.
func formatDec(value *inf.Dec, precision int) string {
	s := value.String()

	if precision == 0 {
		return s
	}

	s = trimTrailingChar(s, '0')
	s = trimTrailingChar(s, '.')

	return s
}

func trimTrailingChar(s string, c byte) string {
	i := len(s)
	for i > 0 && s[i-1] == c {
		i--
	}
	return s[:i]
}
