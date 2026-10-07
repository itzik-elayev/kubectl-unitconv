package convert

import "testing"

func TestResolveTarget(t *testing.T) {
	cases := []struct {
		name    string
		family  Family
		token   string
		want    string
		wantErr bool
	}{
		{name: "real suffix", family: FamilyMemory, token: "Gi", want: "Gi"},
		{name: "bytes alias", family: FamilyMemory, token: "bytes", want: ""},
		{name: "cores alias", family: FamilyCPU, token: "cores", want: ""},
		{name: "millicores alias", family: FamilyCPU, token: "millicores", want: "m"},
		{name: "microcores alias", family: FamilyCPU, token: "microcores", want: "u"},
		{name: "bytes alias on cpu family rejected", family: FamilyCPU, token: "bytes", wantErr: true},
		{name: "cores alias on memory family rejected", family: FamilyMemory, token: "cores", wantErr: true},
		{name: "cpu suffix on memory family rejected", family: FamilyMemory, token: "m", wantErr: true},
		{name: "memory suffix on cpu family rejected", family: FamilyCPU, token: "Mi", wantErr: true},
		{name: "unknown token", family: FamilyMemory, token: "bogus", wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			unit, err := ResolveTarget(c.family, c.token)

			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", unit)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if unit.Suffix != c.want {
				t.Errorf("ResolveTarget(%v, %q) suffix = %q, want %q", c.family, c.token, unit.Suffix, c.want)
			}
		})
	}
}

func TestBytesAndCoresStayDistinctDespiteSharedSuffix(t *testing.T) {
	bytesUnit, err := ResolveTarget(FamilyMemory, "bytes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	coresUnit, err := ResolveTarget(FamilyCPU, "cores")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if bytesUnit.Suffix != coresUnit.Suffix {
		t.Fatalf("expected both to resolve to the empty suffix, got %q and %q", bytesUnit.Suffix, coresUnit.Suffix)
	}
	if bytesUnit.Label == coresUnit.Label {
		t.Error("bytes and cores must not collapse into the same unit despite sharing a suffix")
	}

	// Crucially: a target of "cores" must never be accepted for a memory
	// family quantity, nor "bytes" for a cpu family quantity.
	if _, err := ResolveTarget(FamilyMemory, "cores"); err == nil {
		t.Error("expected error resolving cores target against memory family")
	}
	if _, err := ResolveTarget(FamilyCPU, "bytes"); err == nil {
		t.Error("expected error resolving bytes target against cpu family")
	}
}

func TestParseFamilyFlag(t *testing.T) {
	cases := []struct {
		value      string
		wantFamily Family
		wantForced bool
		wantErr    bool
	}{
		{value: "", wantForced: false},
		{value: "auto", wantForced: false},
		{value: "cpu", wantFamily: FamilyCPU, wantForced: true},
		{value: "Memory", wantFamily: FamilyMemory, wantForced: true},
		{value: "bogus", wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			family, forced, err := ParseFamilyFlag(c.value)

			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", c.value)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if forced != c.wantForced || (forced && family != c.wantFamily) {
				t.Errorf("ParseFamilyFlag(%q) = (%v, %v), want (%v, %v)", c.value, family, forced, c.wantFamily, c.wantForced)
			}
		})
	}
}

func TestResolveFamily(t *testing.T) {
	cases := []struct {
		name       string
		rawInput   string
		fieldHint  string
		targetUnit string
		forced     Family
		hasForced  bool
		want       Family
		wantErr    bool
	}{
		{name: "forced wins with no other signal", rawInput: "2", forced: FamilyCPU, hasForced: true, want: FamilyCPU},
		{name: "field hint wins over suffix guess: memory field with milli suffix", rawInput: "400m", fieldHint: "memory", want: FamilyMemory},
		{name: "field hint cpu", rawInput: "2", fieldHint: "cpu", want: FamilyCPU},
		{name: "field hint storage", rawInput: "2", fieldHint: "storage", want: FamilyMemory},
		{name: "binary suffix implies memory", rawInput: "500Mi", want: FamilyMemory},
		{name: "cpu milli suffix implies cpu with no field context", rawInput: "250m", want: FamilyCPU},
		{name: "ambiguous bare number resolved by explicit target", rawInput: "2", targetUnit: "m", want: FamilyCPU},
		{name: "ambiguous bare number resolved by memory target", rawInput: "2", targetUnit: "Gi", want: FamilyMemory},
		{name: "ambiguous bare number with no signal defaults memory", rawInput: "2", want: FamilyMemory},
		{name: "unambiguous suffix conflicts with target", rawInput: "500Mi", targetUnit: "cores", wantErr: true},
		{name: "forced conflicts with field hint", rawInput: "2", fieldHint: "cpu", forced: FamilyMemory, hasForced: true, wantErr: true},
		{
			name:       "weak milli suffix on literal input yields to an explicit memory target",
			rawInput:   "2387966302991320m", // a real PVC storage value from a live cluster, milli-scale
			targetUnit: "Ti",
			want:       FamilyMemory,
		},
		{
			name:       "weak milli suffix yields to --family memory too",
			rawInput:   "2387966302991320m",
			targetUnit: "Ti",
			forced:     FamilyMemory,
			hasForced:  true,
			want:       FamilyMemory,
		},
		{
			name:      "forced memory overrides a weak suffix guess with no target",
			rawInput:  "400m",
			forced:    FamilyMemory,
			hasForced: true,
			want:      FamilyMemory,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ResolveFamily(c.rawInput, c.fieldHint, c.targetUnit, c.forced, c.hasForced)

			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("ResolveFamily(%q, %q, %q, ...) = %v, want %v", c.rawInput, c.fieldHint, c.targetUnit, got, c.want)
			}
		})
	}
}
