// Package render turns a []result.Result into terminal output: a styled
// table, the original plain-text format, or JSON.
package render

import "fmt"

// OutputMode selects which renderer produces the final output.
type OutputMode string

const (
	OutputAuto  OutputMode = "auto"
	OutputTable OutputMode = "table"
	OutputPlain OutputMode = "plain"
	OutputJSON  OutputMode = "json"
)

// ParseOutputMode validates a --output flag value.
func ParseOutputMode(value string) (OutputMode, error) {
	switch OutputMode(value) {
	case "", OutputAuto:
		return OutputAuto, nil
	case OutputTable, OutputPlain, OutputJSON:
		return OutputMode(value), nil
	default:
		return "", fmt.Errorf("invalid --output value %q: must be auto, table, plain, or json", value)
	}
}

// ResolveOutputMode turns "auto" into a concrete renderer choice based on
// whether stdout is a terminal: a table when it is, the plain format when
// output is redirected (e.g. piped or into a file). Any other mode is
// already concrete and passes through unchanged.
func ResolveOutputMode(mode OutputMode, isTerminal bool) OutputMode {
	if mode != OutputAuto {
		return mode
	}
	if isTerminal {
		return OutputTable
	}
	return OutputPlain
}

// ColorMode selects whether ANSI color is emitted.
type ColorMode string

const (
	ColorAuto   ColorMode = "auto"
	ColorAlways ColorMode = "always"
	ColorNever  ColorMode = "never"
)

// ParseColorMode validates a --color flag value.
func ParseColorMode(value string) (ColorMode, error) {
	switch ColorMode(value) {
	case "", ColorAuto:
		return ColorAuto, nil
	case ColorAlways, ColorNever:
		return ColorMode(value), nil
	default:
		return "", fmt.Errorf("invalid --color value %q: must be auto, always, or never", value)
	}
}

// ResolveColorEnabled decides whether ANSI color should be emitted. Explicit
// --color flags always win; "auto" uses the actual output destination and
// honors NO_COLOR (https://no-color.org: any non-empty value disables color).
func ResolveColorEnabled(mode ColorMode, isTerminal bool, noColorEnv string) bool {
	switch mode {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	default:
		return isTerminal && noColorEnv == ""
	}
}
