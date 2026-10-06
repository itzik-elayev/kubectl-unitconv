package convert

import (
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
		{name: "rounding regression 500M to G", input: "500M", target: "G", family: FamilyMemory, want: "0.5G"},
		{name: "unknown target unit", input: "500Mi", target: "bogus", family: FamilyMemory, wantErr: true},
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

func TestShowAllCPU(t *testing.T) {
	q := resource.MustParse("250m")

	results := ShowAll(q, FamilyCPU, 6)

	want := map[string]string{"n": "250000000n", "m": "250m", "": "0.25"}
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
