package convert

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
)

// Result is one unit/value pair produced by ShowAll.
type Result struct {
	Unit  Unit
	Value string
}

// ToUnit converts a parsed quantity to the requested target unit token
// (a real apimachinery suffix or a friendly alias) and returns the
// formatted value with the target suffix appended, e.g. "0.488281Gi".
//
// Quantity.Value/MilliValue/ScaledValue round up to coarser scales, which is
// correct for k8s resource-limit semantics but wrong for display — so
// conversion goes through AsApproximateFloat64 and plain float64 division
// instead.
func ToUnit(q resource.Quantity, target string, precision int, family Family) (string, error) {
	suffix := ResolveUnitToken(target)

	unit, ok := FindUnit(family, suffix)
	if !ok {
		return "", fmt.Errorf("unknown target unit %q for this quantity", target)
	}

	value := q.AsApproximateFloat64() / unit.Multiplier

	return formatValue(value, precision) + unit.Suffix, nil
}

// ShowAll converts a quantity to every unit in its detected family, in
// ascending order.
func ShowAll(q resource.Quantity, family Family, precision int) []Result {
	units := UnitsFor(family)
	results := make([]Result, 0, len(units))

	base := q.AsApproximateFloat64()

	for _, unit := range units {
		value := formatValue(base/unit.Multiplier, precision)
		results = append(results, Result{Unit: unit, Value: value + unit.Suffix})
	}

	return results
}
