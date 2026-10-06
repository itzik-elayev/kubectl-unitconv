package convert

import "strconv"

// formatValue renders a float64 to at most precision decimal places,
// trimming trailing zeros (and a trailing '.') for a clean display value.
func formatValue(value float64, precision int) string {
	s := strconv.FormatFloat(value, 'f', precision, 64)

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
