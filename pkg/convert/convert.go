package convert

import (
	"gopkg.in/inf.v0"
	"k8s.io/apimachinery/pkg/api/resource"
)

// roundingMode is round-half-to-even (banker's rounding): the IEEE 754
// default, chosen because it has no systematic upward or downward bias
// across many conversions. Converted output is a rounded display value and
// is not guaranteed to round-trip back to the original quantity.
var roundingMode = inf.RoundHalfEven

// Result is one unit/value pair produced by ShowAll.
type Result struct {
	Unit  Unit
	Value string
}

// ToUnit converts a parsed quantity to the requested target unit token
// (a real apimachinery suffix or a friendly alias) and returns the
// formatted value with the target suffix appended, e.g. "0.488281Gi".
//
// Conversion uses exact decimal arithmetic (the quantity's own AsDec, never
// float64), so large integer quantities never lose precision before the
// final, intentional rounding to precision digits.
func ToUnit(q resource.Quantity, target string, precision int, family Family) (string, error) {
	unit, err := ResolveTarget(family, target)
	if err != nil {
		return "", err
	}

	value := new(inf.Dec).QuoRound(q.AsDec(), unit.Multiplier, inf.Scale(precision), roundingMode)

	return formatDec(value, precision) + unit.Suffix, nil
}

// ShowAll converts a quantity to every unit in its family, in ascending
// order, using the same exact decimal arithmetic as ToUnit.
func ShowAll(q resource.Quantity, family Family, precision int) []Result {
	units := UnitsFor(family)
	results := make([]Result, 0, len(units))

	base := q.AsDec()

	for _, unit := range units {
		value := new(inf.Dec).QuoRound(base, unit.Multiplier, inf.Scale(precision), roundingMode)
		results = append(results, Result{Unit: unit, Value: formatDec(value, precision) + unit.Suffix})
	}

	return results
}
