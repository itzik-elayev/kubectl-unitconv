package convert

import (
	"fmt"
	"strings"
)

type familyMismatchError struct {
	what  string // "target unit" or "--family"
	token string
	got   Family
	want  Family
}

func (e *familyMismatchError) Error() string {
	return fmt.Sprintf("%s %q implies %s, which conflicts with %s", e.what, e.token, e.got, e.want)
}

type unknownUnitError struct {
	token  string
	family Family
}

func (e *unknownUnitError) Error() string {
	return fmt.Sprintf("unknown %s target unit %q", e.family, e.token)
}

// ParseFamilyFlag validates a --family flag value, returning the forced
// family and whether one was given ("" or "auto" mean no override).
func ParseFamilyFlag(value string) (family Family, forced bool, err error) {
	switch strings.ToLower(value) {
	case "", "auto":
		return 0, false, nil
	case "cpu":
		return FamilyCPU, true, nil
	case "memory":
		return FamilyMemory, true, nil
	default:
		return 0, false, fmt.Errorf("invalid --family value %q: must be auto, cpu, or memory", value)
	}
}

// familyFromFieldHint reports the family a --from field path's last segment
// name implies. storage/ephemeral-storage are memory-shaped (same unit
// table) but are reported as their own ResourceType by the caller.
func familyFromFieldHint(hint string) (Family, bool) {
	switch strings.ToLower(hint) {
	case "cpu":
		return FamilyCPU, true
	case "memory", "storage", "ephemeral-storage":
		return FamilyMemory, true
	default:
		return 0, false
	}
}

// rawSuffix extracts the trailing non-numeric suffix from a literal quantity
// string, e.g. "500Mi" -> "Mi", "250m" -> "m", "2" -> "".
func rawSuffix(raw string) string {
	i := len(raw)
	for i > 0 {
		c := raw[i-1]
		if c >= '0' && c <= '9' || c == '.' || c == '+' || c == '-' {
			break
		}
		i--
	}
	return raw[i:]
}

// determination tracks the single source of truth for a resolved family, so
// a later disagreeing signal can be reported as a conflict instead of
// silently overwriting an earlier one.
type determination struct {
	family Family
	source string
	set    bool
}

func (d *determination) apply(family Family, source string) error {
	if d.set && d.family != family {
		return fmt.Errorf("%s implies %s, which conflicts with %s (%s)", source, family, d.source, d.family)
	}
	d.family, d.source, d.set = family, source, true
	return nil
}

// ResolveFamily determines whether a quantity is cpu- or memory-shaped.
//
// fieldHint (the last segment of a --from field path) is ground truth when
// present: a resource field's own name is authoritative over guessing from
// the quantity's suffix, since a milli-scale suffix like "400m" is valid on
// a memory quantity too, not just cpu. With no field context (literal CLI
// input), the quantity's own suffix and the requested target unit are used
// instead; an explicit target (e.g. "2" converted to "m") can disambiguate
// an otherwise-ambiguous bare number. A bare number with no other signal
// defaults to memory. Conflicting signals are reported as errors rather
// than silently resolved.
func ResolveFamily(rawInput, fieldHint, targetUnit string, forced Family, hasForced bool) (Family, error) {
	var d determination

	if hasForced {
		if err := d.apply(forced, "--family"); err != nil {
			return 0, err
		}
	}

	if hintFamily, ok := familyFromFieldHint(fieldHint); ok {
		if err := d.apply(hintFamily, "resource field"); err != nil {
			return 0, err
		}
	} else if suffixFamily, ok := familyFromToken(rawSuffix(rawInput)); ok {
		if err := d.apply(suffixFamily, fmt.Sprintf("quantity suffix %q", rawSuffix(rawInput))); err != nil {
			return 0, err
		}
	}

	if targetUnit != "" {
		if targetFamily, ok := familyFromToken(targetUnit); ok {
			if err := d.apply(targetFamily, fmt.Sprintf("target unit %q", targetUnit)); err != nil {
				return 0, err
			}
		}
	}

	if !d.set {
		return FamilyMemory, nil // documented ambiguous default
	}
	return d.family, nil
}
