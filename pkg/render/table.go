package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/muesli/termenv"

	"github.com/itzik-elayev/kubectl-unitconv/pkg/result"
)

type tableStyles struct {
	header     lipgloss.Style
	bold       lipgloss.Style
	boldCell   lipgloss.Style
	accent     lipgloss.Style
	accentCell lipgloss.Style
	muted      lipgloss.Style
	mutedCell  lipgloss.Style
	border     lipgloss.Style
}

// accentColor and mutedColor are adaptive so the table stays readable on
// both light and dark terminal backgrounds.
var (
	accentColor = lipgloss.AdaptiveColor{Light: "#007777", Dark: "#6FFAFA"}
	mutedColor  = lipgloss.AdaptiveColor{Light: "245", Dark: "241"}
)

func newTableStyles(w io.Writer, colorEnabled bool) *tableStyles {
	renderer := lipgloss.NewRenderer(w)
	if colorEnabled {
		renderer.SetColorProfile(termenv.TrueColor)
	} else {
		renderer.SetColorProfile(termenv.Ascii)
	}

	base := renderer.NewStyle()
	cell := base.Padding(0, 1)

	return &tableStyles{
		header:     cell.Bold(true),
		bold:       base.Bold(true),
		boldCell:   cell.Bold(true),
		accent:     base.Foreground(accentColor),
		accentCell: cell.Foreground(accentColor),
		muted:      base.Foreground(mutedColor),
		mutedCell:  cell.Foreground(mutedColor),
		border:     base.Foreground(mutedColor),
	}
}

// Table renders results as a styled table: a compact emphasized line for a
// single context-free conversion, "CONTAINER | RESOURCE | REQUIREMENT |
// ORIGINAL | CONVERTED" rows when fanned out across containers/requirements
// with one target each, or a per-reading unit/value table for show-all mode.
// width is the detected terminal width, or 0 when unknown/not a terminal.
func Table(w io.Writer, results []result.Result, colorEnabled bool, width int) {
	groups := groupResults(results)
	styles := newTableStyles(w, colorEnabled)

	if len(groups) > 1 && allSingleValued(groups) {
		renderFanOut(w, styles, groups, width)
		return
	}

	for i, g := range groups {
		if i > 0 {
			fmt.Fprintln(w)
		}
		renderGroup(w, styles, g, width)
	}
}

func allSingleValued(groups []group) bool {
	for _, g := range groups {
		if len(g.values) != 1 {
			return false
		}
	}
	return true
}

func renderGroup(w io.Writer, styles *tableStyles, g group, width int) {
	if len(g.values) == 1 {
		renderCompact(w, styles, g)
		return
	}
	renderShowAll(w, styles, g, width)
}

// renderCompact prints "<original> (<converted>)", converted emphasized —
// the presentation for a single literal-to-one-target conversion.
func renderCompact(w io.Writer, styles *tableStyles, g group) {
	line := fmt.Sprintf("%s (%s)",
		styles.muted.Render(g.context.Original),
		styles.accent.Bold(true).Render(g.values[0].converted))

	if label := readingLabel(g.context); label != "" {
		line = styles.bold.Render(label) + ": " + line
	}

	fmt.Fprintln(w, line)
}

func renderShowAll(w io.Writer, styles *tableStyles, g group, width int) {
	if caption := captionFor(g.context); caption != "" {
		fmt.Fprintln(w, styles.muted.Render(caption))
	}

	t := table.New().
		BorderStyle(styles.border).
		Headers("UNIT", "VALUE").
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return styles.header
			case col == 1:
				return styles.accentCell
			default:
				return styles.mutedCell
			}
		})
	if width > 0 {
		t.Width(width)
	}

	for _, v := range g.values {
		t.Row(v.label, v.converted)
	}

	fmt.Fprintln(w, t.Render())
}

func renderFanOut(w io.Writer, styles *tableStyles, groups []group, width int) {
	t := table.New().
		BorderStyle(styles.border).
		Headers("CONTAINER", "RESOURCE", "REQUIREMENT", "ORIGINAL", "CONVERTED").
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return styles.header
			case col == 0:
				return styles.boldCell
			case col == 4:
				return styles.accentCell
			default:
				return styles.mutedCell
			}
		})
	if width > 0 {
		t.Width(width)
	}

	for _, g := range groups {
		container := g.context.Container
		if container == "" {
			container = "-"
		} else if g.context.IsInitContainer {
			container += " (init)"
		}

		requirement := g.context.Requirement
		if requirement == "" {
			requirement = "-"
		}

		t.Row(container, string(g.context.ResourceType), requirement, g.context.Original, g.values[0].converted)
	}

	fmt.Fprintln(w, t.Render())
}

// captionFor summarizes a group's shared context once, so a show-all table
// doesn't repeat it on every row.
func captionFor(ctx result.Result) string {
	var parts []string

	if ctx.Kind != "" && ctx.Name != "" {
		ref := ctx.Kind + "/" + ctx.Name
		if ctx.Namespace != "" {
			ref = ctx.Namespace + "/" + ref
		}
		parts = append(parts, ref)
	}

	if label := readingLabel(ctx); label != "" {
		parts = append(parts, label)
	}

	parts = append(parts, "original "+ctx.Original)

	return strings.Join(parts, " · ")
}
