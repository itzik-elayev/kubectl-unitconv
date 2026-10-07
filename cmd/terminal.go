package cmd

import (
	"io"
	"os"

	"github.com/charmbracelet/x/term"
)

// isTerminalWriter reports whether w is a terminal, so "auto" output/color
// modes can key off the actual destination rather than assuming stdout.
func isTerminalWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(f.Fd())
}

// terminalWidth returns w's terminal width, or 0 when it isn't a terminal or
// the size can't be determined — callers treat 0 as "unconstrained".
func terminalWidth(w io.Writer, isTerminal bool) int {
	if !isTerminal {
		return 0
	}
	f, ok := w.(*os.File)
	if !ok {
		return 0
	}
	width, _, err := term.GetSize(f.Fd())
	if err != nil {
		return 0
	}
	return width
}
