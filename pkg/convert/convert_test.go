package convert

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
)

func TestToUnit(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		target  string
		family  Family
		want    string
		wantErr bool
	}{
		{name: "binary mi to gi", input: "500Mi", target: "Gi", family: FamilyMemory, want: "0.488281Gi"},
		{name: "decimal bytes to gi", input: "2000000000", target: "Gi", family: FamilyMemory, want: "1.862645Gi"},
		{name: "cpu milli to cores", input: "250m", target: "cores", family: FamilyCPU, want: "0.25"},
		{name: "cpu cores to nano", input: "2", target: "n", family: FamilyCPU, want: "2000000000n"},
		{name: "cpu cores to microcores", input: "2", target: "u", family: FamilyCPU, want: "2000000u"},
		{name: "cpu cores to microcores alias", input: "2", target: "microcores", family: FamilyCPU, want: "2000000u"},
		{name: "rounding regression 500M to G", input: "500M", target: "G", family: FamilyMemory, want: "0.5G"},
		{name: "unknown target unit", input: "500Mi", target: "bogus", family: FamilyMemory, wantErr: true},
		{name: "cores target on memory family rejected", input: "500Mi", target: "cores", family: FamilyMemory, wantErr: true},
		{name: "bytes target on cpu family rejected", input: "2", target: "bytes", family: FamilyCPU, wantErr: true},
		{name: "Gi target on cpu family rejected", input: "2", target: "Gi", family: FamilyCPU, wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q := resource.MustParse(c.input)

			got, err := ToUnit(q, c.target, 6, c.family)

			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("ToUnit(%q, %q) = %q, want %q", c.input, c.target, got, c.want)
			}
		})
	}
}

func TestToUnitLargeValuesExactPrecision(t *testing.T) {
	// 2^63-1 bytes: AsApproximateFloat64 would lose precision here (float64
	// has only ~15-17 significant decimal digits); exact decimal must not.
	q := resource.MustParse("9223372036854775807")

	got, err := ToUnit(q, "bytes", 0, FamilyMemory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "9223372036854775807" {
		t.Errorf("got %q, want exact round-trip of a huge integer", got)
	}
}

func TestToUnitSmallNonzeroQuantity(t *testing.T) {
	// 1 nanocore is far below float64's useful precision once scaled by a
	// multiplier; exact decimal must still resolve it correctly at a high
	// enough precision, and round it to zero at a low one.
	q := resource.MustParse("1n")

	got, err := ToUnit(q, "cores", 9, FamilyCPU)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "0.000000001" {
		t.Errorf("got %q, want 0.000000001", got)
	}

	got, err = ToUnit(q, "cores", 6, FamilyCPU)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "0" {
		t.Errorf("got %q, want 0 (rounded below display precision)", got)
	}
}

func TestToUnitRoundingBoundaryHalfToEven(t *testing.T) {
	// 0.125 rounded to 2 decimal places is exactly on the half-way point
	// (0.125 -> 0.12 or 0.13). Round-half-to-even rounds to the nearest even
	// last digit: 0.12.
	q := resource.MustParse("125m")

	got, err := ToUnit(q, "cores", 2, FamilyCPU)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "0.12" {
		t.Errorf("got %q, want 0.12 (round-half-to-even)", got)
	}

	// 0.375 -> 2 decimals: half-way between 0.37 and 0.38; 0.38 is even.
	q = resource.MustParse("375m")
	got, err = ToUnit(q, "cores", 2, FamilyCPU)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "0.38" {
		t.Errorf("got %q, want 0.38 (round-half-to-even)", got)
	}
}

func TestToUnitPrecisionZeroDoesNotTrimIntegerDigits(t *testing.T) {
	q := resource.MustParse("100Mi")

	got, err := ToUnit(q, "Ki", 0, FamilyMemory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "102400Ki" {
		t.Errorf("got %q, want 102400Ki (trailing zeros in an integer must survive precision=0)", got)
	}
}

func TestParseQuantityRejectsUppercaseK(t *testing.T) {
	if _, err := resource.ParseQuantity("1K"); err == nil {
		t.Fatal("expected ParseQuantity(\"1K\") to error, decimal SI suffix is lowercase k only")
	}
}

func TestShowAllMemory(t *testing.T) {
	q := resource.MustParse("500Mi")

	results := ShowAll(q, FamilyMemory, 6)

	if len(results) != len(MemoryUnits) {
		t.Fatalf("got %d results, want %d", len(results), len(MemoryUnits))
	}

	want := map[string]string{"": "524288000", "Mi": "500Mi", "Gi": "0.488281Gi"}
	got := map[string]string{}
	for i, r := range results {
		got[MemoryUnits[i].Suffix] = r.Value
	}
	for suffix, value := range want {
		if got[suffix] != value {
			t.Errorf("suffix %q = %q, want %q", suffix, got[suffix], value)
		}
	}
}

func TestShowAllMarksZeroValues(t *testing.T) {
	// Relevance is checked at a fixed 2 decimal places regardless of the
	// requested display precision (6 here): T/Ti/P/Pi/E/Ei all round to
	// 0.00 for 500Mi (needing 3+ leading zeros to show any digit), even
	// though some of them (e.g. petabytes: 0.000524) are technically
	// nonzero at the display precision.
	q := resource.MustParse("500Mi")

	results := ShowAll(q, FamilyMemory, 6)

	wantZero := map[string]bool{
		"": false, "k": false, "Ki": false, "M": false, "Mi": false, "G": false, "Gi": false,
		"T": true, "Ti": true, "P": true, "Pi": true, "E": true, "Ei": true,
	}
	for i, r := range results {
		suffix := MemoryUnits[i].Suffix
		if want, ok := wantZero[suffix]; ok && r.IsZero != want {
			t.Errorf("suffix %q IsZero = %v, want %v", suffix, r.IsZero, want)
		}
	}
}

func TestShowAllRelevanceIgnoresTinyNonzeroValues(t *testing.T) {
	// A value that needs several leading zeros to show any digit is noise
	// regardless of how much display --precision reveals it at.
	q := resource.MustParse("18394417215832064m") // ~18.4 Ti

	for _, precision := range []int{2, 6, 18} {
		results := ShowAll(q, FamilyMemory, precision)
		for i, r := range results {
			suffix := MemoryUnits[i].Suffix
			wantZero := suffix == "E" || suffix == "Ei"
			if r.IsZero != wantZero {
				t.Errorf("precision=%d suffix %q IsZero = %v, want %v", precision, suffix, r.IsZero, wantZero)
			}
		}
	}
}

func TestShowAllZeroInputMarksEveryUnitZero(t *testing.T) {
	q := resource.MustParse("0")

	for _, r := range ShowAll(q, FamilyMemory, 6) {
		if !r.IsZero {
			t.Errorf("unit %q: IsZero = false, want true for a zero quantity", r.Unit.Suffix)
		}
	}
}

func TestShowAllCPUIncludesMicrocores(t *testing.T) {
	q := resource.MustParse("250m")

	results := ShowAll(q, FamilyCPU, 6)

	want := map[string]string{"n": "250000000n", "u": "250000u", "m": "250m", "": "0.25"}
	if len(results) != len(want) {
		t.Fatalf("got %d results, want %d (n, u, m, cores)", len(results), len(want))
	}
	got := map[string]string{}
	for i, r := range results {
		got[CPUUnits[i].Suffix] = r.Value
	}
	for suffix, value := range want {
		if got[suffix] != value {
			t.Errorf("suffix %q = %q, want %q", suffix, got[suffix], value)
		}
	}
}

func TestValidatePrecision(t *testing.T) {
	if err := ValidatePrecision(-1); err == nil {
		t.Error("expected error for negative precision")
	}
	if err := ValidatePrecision(MaxPrecision + 1); err == nil {
		t.Error("expected error for precision above the documented bound")
	}
	if err := ValidatePrecision(0); err != nil {
		t.Errorf("precision 0 should be valid: %v", err)
	}
	if err := ValidatePrecision(MaxPrecision); err != nil {
		t.Errorf("precision at the max bound should be valid: %v", err)
	}
}

func TestRoundedOutputDoesNotClaimRoundTrip(t *testing.T) {
	// Documents (via behavior) that a rounded display value is lossy: 1Mi
	// shown at low precision in Gi, then re-parsed, is not exactly 1Mi.
	q := resource.MustParse("1Mi")

	rounded, err := ToUnit(q, "Gi", 2, FamilyMemory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rounded != "0Gi" {
		t.Fatalf("got %q, want 0Gi", rounded)
	}

	reparsed := resource.MustParse(strings.TrimSuffix(rounded, "Gi"))
	if reparsed.Cmp(q) == 0 {
		t.Fatal("rounded display value unexpectedly round-tripped exactly; it should not be relied on to")
	}
}
