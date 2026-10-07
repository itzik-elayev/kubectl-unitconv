package render

import (
	"github.com/itzik-elayev/kubectl-unitconv/pkg/convert"
	"github.com/itzik-elayev/kubectl-unitconv/pkg/result"
)

// unitValue is one target unit/converted-value pair within a group.
type unitValue struct {
	label     string // descriptive label, e.g. "kibibytes (2^10)"
	converted string
}

// group is one underlying quantity (a single reading or literal input)
// together with every target-unit conversion produced for it. Results
// sharing identical context collapse into one group so renderers can print
// shared metadata once instead of once per unit — the point of grouping at
// all, since a flat []Result repeats that context per row by design (it's
// the stable, flat shape JSON consumers want).
type group struct {
	context result.Result // TargetUnit/Converted on this value are unused
	values  []unitValue
}

// groupResults collapses a flat []Result into groups by shared context,
// preserving input order (stable for both groups and values within a group).
func groupResults(results []result.Result) []group {
	var groups []group
	index := map[string]int{}

	for _, r := range results {
		key := groupKey(r)
		label := unitLabel(r)

		if i, ok := index[key]; ok {
			groups[i].values = append(groups[i].values, unitValue{label: label, converted: r.Converted})
			continue
		}

		index[key] = len(groups)
		groups = append(groups, group{
			context: r,
			values:  []unitValue{{label: label, converted: r.Converted}},
		})
	}

	return groups
}

func groupKey(r result.Result) string {
	return r.Kind + "\x00" + r.Name + "\x00" + r.Namespace + "\x00" + r.Container + "\x00" +
		boolString(r.IsInitContainer) + "\x00" + string(r.ResourceType) + "\x00" + r.Requirement + "\x00" + r.Original
}

func boolString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// unitLabel looks up the descriptive label for a result's target unit
// (e.g. "kibibytes (2^10)"), re-derived from the unit tables rather than
// stored on Result itself, which only carries the canonical suffix.
func unitLabel(r result.Result) string {
	if unit, ok := convert.FindUnit(r.ResourceType.Family(), r.TargetUnit); ok {
		return unit.Label
	}
	return r.TargetUnit
}

// readingLabel describes which container/requirement a group's values came
// from, or "" for a context-free reading (a literal conversion, or a single
// resource with no container concept such as a PVC or node).
func readingLabel(ctx result.Result) string {
	if ctx.Container == "" {
		return ""
	}

	kind := "container"
	if ctx.IsInitContainer {
		kind = "init container"
	}
	if ctx.Requirement == "" {
		return kind + " " + ctx.Container
	}
	return kind + " " + ctx.Container + " (" + ctx.Requirement + ")"
}
