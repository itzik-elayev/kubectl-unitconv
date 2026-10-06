package convert

import "strings"

// Family distinguishes which unit table a quantity belongs to, since
// resource.Quantity itself carries no semantic tag for cpu vs memory.
type Family int

const (
	FamilyMemory Family = iota
	FamilyCPU
)

// Unit is one named point in a Family's conversion table.
type Unit struct {
	Suffix     string // apimachinery-valid suffix; "" = base unit
	Label      string
	Multiplier float64 // value of 1 of this unit, expressed in the family's base unit
}

const (
	kibi = 1 << 10
	mebi = 1 << 20
	gibi = 1 << 30
	tebi = 1 << 40
	pebi = 1 << 50
	exbi = 1 << 60
)

// MemoryUnits lists every unit shown by show-all mode for memory-like
// quantities, in ascending order. Base unit is bytes.
var MemoryUnits = []Unit{
	{"", "bytes", 1},
	{"k", "kilobytes (10^3)", 1e3},
	{"Ki", "kibibytes (2^10)", kibi},
	{"M", "megabytes (10^6)", 1e6},
	{"Mi", "mebibytes (2^20)", mebi},
	{"G", "gigabytes (10^9)", 1e9},
	{"Gi", "gibibytes (2^30)", gibi},
	{"T", "terabytes (10^12)", 1e12},
	{"Ti", "tebibytes (2^40)", tebi},
	{"P", "petabytes (10^15)", 1e15},
	{"Pi", "pebibytes (2^50)", pebi},
	{"E", "exabytes (10^18)", 1e18},
	{"Ei", "exbibytes (2^60)", exbi},
}

// CPUUnits lists every unit shown by show-all mode for cpu-like quantities,
// in ascending order. Base unit is cores.
var CPUUnits = []Unit{
	{"n", "nanocores", 1e-9},
	{"m", "millicores", 1e-3},
	{"", "cores", 1},
}

// unitAliases maps user-friendly target tokens to their real apimachinery
// suffix. Matched case-insensitively before falling back to an exact-case
// match against a real k8s suffix (e.g. "Gi" vs "gi" are not interchangeable
// to apimachinery itself).
var unitAliases = map[string]string{
	"bytes":      "",
	"byte":       "",
	"b":          "",
	"cores":      "",
	"core":       "",
	"millicores": "m",
	"nanocores":  "n",
	"microcores": "u",
}

// ResolveUnitToken turns a user-supplied target unit token into the real
// apimachinery suffix to use, applying the alias table and binary/decimal
// suffix casing rules.
func ResolveUnitToken(token string) string {
	if suffix, ok := unitAliases[strings.ToLower(token)]; ok {
		return suffix
	}
	return token
}

// UnitsFor returns the show-all unit table for a Family.
func UnitsFor(family Family) []Unit {
	if family == FamilyCPU {
		return CPUUnits
	}
	return MemoryUnits
}

// FindUnit looks up a resolved suffix within a Family's unit table.
func FindUnit(family Family, suffix string) (Unit, bool) {
	for _, u := range UnitsFor(family) {
		if u.Suffix == suffix {
			return u, true
		}
	}
	return Unit{}, false
}

// binarySuffixes and cpuSuffixes drive DetectFamily — each is unambiguous on
// its own, unlike decimal SI suffixes which are shared between interpretations.
var binarySuffixes = map[string]bool{"Ki": true, "Mi": true, "Gi": true, "Ti": true, "Pi": true, "Ei": true}
var cpuSuffixes = map[string]bool{"n": true, "u": true, "m": true}

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

// DetectFamily picks which unit family a quantity belongs to, in priority
// order: an explicit --as override, a field-path hint from --from, then the
// literal input's own suffix. Ambiguous suffixes (bare number, decimal SI)
// default to memory.
func DetectFamily(rawInput, fieldHint, forcedAs string) Family {
	switch strings.ToLower(forcedAs) {
	case "cpu":
		return FamilyCPU
	case "memory":
		return FamilyMemory
	}

	switch strings.ToLower(fieldHint) {
	case "cpu":
		return FamilyCPU
	case "memory", "storage", "ephemeral-storage":
		return FamilyMemory
	}

	suffix := rawSuffix(rawInput)
	if binarySuffixes[suffix] {
		return FamilyMemory
	}
	if cpuSuffixes[suffix] {
		return FamilyCPU
	}

	return FamilyMemory
}
