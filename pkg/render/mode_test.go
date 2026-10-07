package render

import "testing"

func TestParseOutputMode(t *testing.T) {
	cases := []struct {
		value   string
		want    OutputMode
		wantErr bool
	}{
		{value: "", want: OutputAuto},
		{value: "auto", want: OutputAuto},
		{value: "table", want: OutputTable},
		{value: "plain", want: OutputPlain},
		{value: "json", want: OutputJSON},
		{value: "bogus", wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			got, err := ParseOutputMode(c.value)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", c.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("ParseOutputMode(%q) = %q, want %q", c.value, got, c.want)
			}
		})
	}
}

func TestResolveOutputMode(t *testing.T) {
	cases := []struct {
		name       string
		mode       OutputMode
		isTerminal bool
		want       OutputMode
	}{
		{name: "auto on a terminal resolves to table", mode: OutputAuto, isTerminal: true, want: OutputTable},
		{name: "auto when redirected resolves to plain", mode: OutputAuto, isTerminal: false, want: OutputPlain},
		{name: "explicit table survives redirection", mode: OutputTable, isTerminal: false, want: OutputTable},
		{name: "explicit json survives a terminal", mode: OutputJSON, isTerminal: true, want: OutputJSON},
		{name: "explicit plain stays plain on a terminal", mode: OutputPlain, isTerminal: true, want: OutputPlain},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ResolveOutputMode(c.mode, c.isTerminal)
			if got != c.want {
				t.Errorf("ResolveOutputMode(%q, %v) = %q, want %q", c.mode, c.isTerminal, got, c.want)
			}
		})
	}
}

func TestParseColorMode(t *testing.T) {
	cases := []struct {
		value   string
		want    ColorMode
		wantErr bool
	}{
		{value: "", want: ColorAuto},
		{value: "auto", want: ColorAuto},
		{value: "always", want: ColorAlways},
		{value: "never", want: ColorNever},
		{value: "bogus", wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			got, err := ParseColorMode(c.value)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", c.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("ParseColorMode(%q) = %q, want %q", c.value, got, c.want)
			}
		})
	}
}

func TestResolveColorEnabled(t *testing.T) {
	cases := []struct {
		name       string
		mode       ColorMode
		isTerminal bool
		noColorEnv string
		want       bool
	}{
		{name: "always wins even when redirected", mode: ColorAlways, isTerminal: false, want: true},
		{name: "always wins even with NO_COLOR set", mode: ColorAlways, isTerminal: true, noColorEnv: "1", want: true},
		{name: "never wins even on a terminal", mode: ColorNever, isTerminal: true, want: false},
		{name: "auto on a terminal enables color", mode: ColorAuto, isTerminal: true, want: true},
		{name: "auto when redirected disables color", mode: ColorAuto, isTerminal: false, want: false},
		{name: "auto honors NO_COLOR on a terminal", mode: ColorAuto, isTerminal: true, noColorEnv: "1", want: false},
		{name: "auto with NO_COLOR empty string still colors a terminal", mode: ColorAuto, isTerminal: true, noColorEnv: "", want: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ResolveColorEnabled(c.mode, c.isTerminal, c.noColorEnv)
			if got != c.want {
				t.Errorf("ResolveColorEnabled(%q, %v, %q) = %v, want %v", c.mode, c.isTerminal, c.noColorEnv, got, c.want)
			}
		})
	}
}
