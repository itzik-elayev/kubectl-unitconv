package convert

import "testing"

func TestDetectFamily(t *testing.T) {
	cases := []struct {
		name      string
		rawInput  string
		fieldHint string
		forcedAs  string
		want      Family
	}{
		{name: "forced as wins over everything", rawInput: "500Mi", fieldHint: "cpu", forcedAs: "memory", want: FamilyMemory},
		{name: "field hint cpu", rawInput: "2", fieldHint: "cpu", want: FamilyCPU},
		{name: "field hint memory", rawInput: "2", fieldHint: "memory", want: FamilyMemory},
		{name: "field hint storage", rawInput: "2", fieldHint: "storage", want: FamilyMemory},
		{name: "binary suffix", rawInput: "500Mi", want: FamilyMemory},
		{name: "cpu milli suffix", rawInput: "250m", want: FamilyCPU},
		{name: "cpu nano suffix", rawInput: "250n", want: FamilyCPU},
		{name: "ambiguous bare number defaults memory", rawInput: "2", want: FamilyMemory},
		{name: "decimal SI defaults memory", rawInput: "2G", want: FamilyMemory},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DetectFamily(c.rawInput, c.fieldHint, c.forcedAs)
			if got != c.want {
				t.Errorf("DetectFamily(%q, %q, %q) = %v, want %v", c.rawInput, c.fieldHint, c.forcedAs, got, c.want)
			}
		})
	}
}

func TestResolveUnitToken(t *testing.T) {
	cases := map[string]string{
		"bytes":      "",
		"Bytes":      "",
		"cores":      "",
		"millicores": "m",
		"nanocores":  "n",
		"microcores": "u",
		"Gi":         "Gi",
		"m":          "m",
	}

	for token, want := range cases {
		if got := ResolveUnitToken(token); got != want {
			t.Errorf("ResolveUnitToken(%q) = %q, want %q", token, got, want)
		}
	}
}
