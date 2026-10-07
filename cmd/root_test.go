package cmd

import (
	"bytes"
	"strings"
	"testing"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func newTestOptions() *rootOptions {
	return &rootOptions{family: "auto", output: "auto", color: "auto", precision: 6}
}

func runForTest(args []string, opts *rootOptions) (string, error) {
	var buf bytes.Buffer
	err := runRoot(&buf, args, opts, genericclioptions.NewConfigFlags(true))
	return buf.String(), err
}

func TestRunRootLiteralConversion(t *testing.T) {
	out, err := runForTest([]string{"500Mi", "Gi"}, newTestOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// A non-terminal buffer resolves "auto" output to plain: a bare value.
	if out != "0.488281Gi\n" {
		t.Errorf("got %q, want %q", out, "0.488281Gi\n")
	}
}

func TestRunRootLiteralShowAllPreservesOriginal(t *testing.T) {
	out, err := runForTest([]string{"2", "m"}, newTestOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "2000m\n" {
		t.Errorf("got %q, want %q (explicit target should resolve the ambiguous bare number to cpu)", out, "2000m\n")
	}
}

func TestRunRootInvalidFamily(t *testing.T) {
	opts := newTestOptions()
	opts.family = "bogus"
	if _, err := runForTest([]string{"2"}, opts); err == nil {
		t.Fatal("expected error for invalid --family")
	}
}

func TestRunRootInvalidOutput(t *testing.T) {
	opts := newTestOptions()
	opts.output = "bogus"
	if _, err := runForTest([]string{"2"}, opts); err == nil {
		t.Fatal("expected error for invalid --output")
	}
}

func TestRunRootInvalidColor(t *testing.T) {
	opts := newTestOptions()
	opts.color = "bogus"
	if _, err := runForTest([]string{"2"}, opts); err == nil {
		t.Fatal("expected error for invalid --color")
	}
}

func TestRunRootNegativePrecisionRejected(t *testing.T) {
	opts := newTestOptions()
	opts.precision = -1
	if _, err := runForTest([]string{"2"}, opts); err == nil {
		t.Fatal("expected error for negative --precision")
	}
}

func TestRunRootContainerFlagRequiresFrom(t *testing.T) {
	opts := newTestOptions()
	opts.container = "app"
	_, err := runForTest([]string{"2", "Gi"}, opts)
	if err == nil {
		t.Fatal("expected error: --container without --from")
	}
	if !strings.Contains(err.Error(), "--from") {
		t.Errorf("error should mention --from, got: %v", err)
	}
}

func TestRunRootRequirementFlagRejectedWithExplicitFromPath(t *testing.T) {
	opts := newTestOptions()
	opts.from = "pod/x:spec.containers[0].resources.requests.memory"
	opts.requirement = "limits"
	_, err := runForTest(nil, opts)
	if err == nil {
		t.Fatal("expected error: --requirement does not apply to an explicit --from field path")
	}
}

func TestRunRootInvalidRequirementValue(t *testing.T) {
	opts := newTestOptions()
	opts.from = "pod/x"
	opts.requirement = "bogus"
	_, err := runForTest(nil, opts)
	if err == nil {
		t.Fatal("expected error for invalid --requirement value")
	}
}

func TestRunRootMissingValueWithoutFrom(t *testing.T) {
	if _, err := runForTest(nil, newTestOptions()); err == nil {
		t.Fatal("expected error when no VALUE and no --from are given")
	}
}

func TestRunRootJSONOutputIsValidAndUncolored(t *testing.T) {
	opts := newTestOptions()
	opts.output = "json"
	out, err := runForTest([]string{"500Mi", "Gi"}, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "\x1b[") {
		t.Error("json output must never contain ANSI escape codes")
	}
	if !strings.Contains(out, `"original": "500Mi"`) {
		t.Errorf("expected original value preserved in JSON output, got: %s", out)
	}
}
