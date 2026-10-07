package convert

import (
	"strings"

	"gopkg.in/inf.v0"
)

// Family distinguishes which unit table a quantity belongs to, since
// resource.Quantity itself carries no semantic tag for cpu vs memory.
type Family int

const (
	FamilyMemory Family = iota
	FamilyCPU
)

func (f Family) String() string {
	if f == FamilyCPU {
		return "cpu"
	}
	return "memory"
}

// Unit is one named point in a Family's conversion table. Multiplier is
// exact (not float64) so conversions never lose precision on large values.
type Unit struct {
	Suffix     string // apimachinery-valid suffix; "" = base unit
	Label      string
	Multiplier *inf.Dec // value of 1 of this unit, expressed in the family's base unit
}

// decPow10 returns an exact decimal for 10^exp.
func decPow10(exp int) *inf.Dec {
	return inf.NewDec(1, inf.Scale(-exp))
}

// decInt returns an exact decimal for a whole number.
func decInt(n int64) *inf.Dec {
	return inf.NewDec(n, 0)
}

// MemoryUnits lists every unit shown by show-all mode for memory-like
// quantities, in ascending order. Base unit is bytes.
var MemoryUnits = []Unit{
	{"", "bytes", decInt(1)},
	{"k", "kilobytes (10^3)", decPow10(3)},
	{"Ki", "kibibytes (2^10)", decInt(1 << 10)},
	{"M", "megabytes (10^6)", decPow10(6)},
	{"Mi", "mebibytes (2^20)", decInt(1 << 20)},
	{"G", "gigabytes (10^9)", decPow10(9)},
	{"Gi", "gibibytes (2^30)", decInt(1 << 30)},
	{"T", "terabytes (10^12)", decPow10(12)},
	{"Ti", "tebibytes (2^40)", decInt(1 << 40)},
	{"P", "petabytes (10^15)", decPow10(15)},
	{"Pi", "pebibytes (2^50)", decInt(1 << 50)},
	{"E", "exabytes (10^18)", decPow10(18)},
	{"Ei", "exbibytes (2^60)", decInt(1 << 60)},
}

// CPUUnits lists every unit shown by show-all mode for cpu-like quantities,
// in ascending order. Base unit is cores.
var CPUUnits = []Unit{
	{"n", "nanocores", decPow10(-9)},
	{"u", "microcores", decPow10(-6)},
	{"m", "millicores", decPow10(-3)},
	{"", "cores", decInt(1)},
}

// unitRef names a unit by family + suffix. Aliases resolve to a unitRef
// rather than a bare suffix so that tokens sharing a suffix across families
// (bytes vs cores, both ""), never collapse into one another.
type unitRef struct {
	family Family
	suffix string
}

// aliasTable maps a friendly token to the specific unit it names. Resolved
// up front (rather than via suffix guessing) so "bytes" and "cores" stay
// semantically distinct despite sharing the empty real suffix.
var aliasTable = map[string]unitRef{
	"bytes":      {FamilyMemory, ""},
	"byte":       {FamilyMemory, ""},
	"b":          {FamilyMemory, ""},
	"cores":      {FamilyCPU, ""},
	"core":       {FamilyCPU, ""},
	"millicores": {FamilyCPU, "m"},
	"nanocores":  {FamilyCPU, "n"},
	"microcores": {FamilyCPU, "u"},
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

// ResolveTarget turns a user-supplied target unit token (a real apimachinery
// suffix, or a friendly alias) into the unit to convert to for the given
// family. Returns an error if the token names a unit from the other family
// (e.g. target "cores" with a memory quantity).
func ResolveTarget(family Family, token string) (Unit, error) {
	if ref, ok := aliasTable[strings.ToLower(token)]; ok {
		if ref.family != family {
			return Unit{}, &familyMismatchError{what: "target unit", token: token, got: ref.family, want: family}
		}
		unit, _ := FindUnit(ref.family, ref.suffix)
		return unit, nil
	}

	if unit, ok := FindUnit(family, token); ok {
		return unit, nil
	}

	if other := otherFamily(family); isUnitSuffix(other, token) {
		return Unit{}, &familyMismatchError{what: "target unit", token: token, got: other, want: family}
	}

	return Unit{}, &unknownUnitError{token: token, family: family}
}

func isUnitSuffix(family Family, suffix string) bool {
	if suffix == "" {
		return false // "" is the base-unit suffix for both families; not a distinguishing signal
	}
	_, ok := FindUnit(family, suffix)
	return ok
}

func otherFamily(family Family) Family {
	if family == FamilyCPU {
		return FamilyMemory
	}
	return FamilyCPU
}

// FamilyFromTargetUnit reports the family a target-unit token unambiguously
// implies, via the alias table or a suffix unique to one family. The empty
// suffix is inherently ambiguous (bytes vs cores) and never reported here.
// Exported for callers (such as a --from default-path lookup) that must
// pick a family from a target unit alone, with no quantity suffix to guess
// from.
func FamilyFromTargetUnit(token string) (Family, bool) {
	return familyFromToken(token)
}

func familyFromToken(token string) (Family, bool) {
	if ref, ok := aliasTable[strings.ToLower(token)]; ok {
		return ref.family, true
	}
	if isUnitSuffix(FamilyMemory, token) {
		return FamilyMemory, true
	}
	if isUnitSuffix(FamilyCPU, token) {
		return FamilyCPU, true
	}
	return 0, false
}
