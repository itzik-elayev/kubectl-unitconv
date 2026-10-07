package render

import (
	"fmt"
	"io"

	"github.com/itzik-elayev/kubectl-unitconv/pkg/result"
)

// Plain renders results as uncolored text: a bare converted value for a
// single literal conversion, a "<label>: <value>" line per --from reading
// when a target unit was given, or a padded unit/value list for show-all
// mode (indented and labeled when it came from a --from reading).
func Plain(w io.Writer, results []result.Result) {
	for _, g := range groupResults(results) {
		label := readingLabel(g.context)

		if len(g.values) == 1 {
			printSingle(w, label, g.values[0].converted)
			continue
		}

		printList(w, label, g.values)
	}
}

func printSingle(w io.Writer, label, converted string) {
	if label == "" {
		fmt.Fprintln(w, converted)
		return
	}
	fmt.Fprintf(w, "%s: %s\n", label, converted)
}

func printList(w io.Writer, label string, values []unitValue) {
	indent := ""
	if label != "" {
		fmt.Fprintf(w, "%s:\n", label)
		indent = "  "
	}
	for _, v := range values {
		fmt.Fprintf(w, "%s%-20s %s\n", indent, v.label, v.converted)
	}
}
