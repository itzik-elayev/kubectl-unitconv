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

func TestRunRootMilliScaleStorageValueConvertsAsMemory(t *testing.T) {
	// A real PVC storage value from a live cluster, written in milli-scale —
	// apimachinery allows this on any quantity, not just cpu. The explicit
	// memory-only target unit must decide family, not the "m" suffix.
	out, err := runForTest([]string{"2387966302991320m", "Ti"}, newTestOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "2.171843Ti\n" {
		t.Errorf("got %q, want %q", out, "2.171843Ti\n")
	}

	opts := newTestOptions()
	opts.family = "memory"
	out, err = runForTest([]string{"2387966302991320m", "Ti"}, opts)
	if err != nil {
		t.Fatalf("unexpected error with --family memory: %v", err)
	}
	if out != "2.171843Ti\n" {
		t.Errorf("got %q, want %q", out, "2.171843Ti\n")
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

func TestRunRootShowAllDropsUnitsBelowOne(t *testing.T) {
	out, err := runForTest([]string{"500Mi"}, newTestOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 500Mi is under one whole unit from gigabytes upward (0.488281Gi).
	for _, unwanted := range []string{"gibibytes", "gigabytes", "terabytes", "tebibytes", "petabytes", "pebibytes", "exabytes", "exbibytes"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("output should drop units the quantity is below one of, found %q in:\n%s", unwanted, out)
		}
	}
	if !strings.Contains(out, "500Mi") || !strings.Contains(out, "524.288M") {
		t.Errorf("expected mebibytes/megabytes rows preserved, got:\n%s", out)
	}
}

func TestRunRootShowAllOrdersLargestFirst(t *testing.T) {
	out, err := runForTest([]string{"500Mi"}, newTestOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Every label ends in "...bytes" (kilobytes, kibibytes, ...), so anchor
	// on each one's distinguishing scale marker instead of the plain word;
	// the standalone "bytes" row is the only one starting a line with it.
	mebi := strings.Index(out, "(2^20)")
	kibi := strings.Index(out, "(2^10)")
	plainBytes := strings.Index(out, "\nbytes ")

	if mebi < 0 || !(mebi < kibi && kibi < plainBytes) {
		t.Errorf("expected largest-to-smallest order (Mi, Ki, bytes), got:\n%s", out)
	}
}

func TestRunRootShowAllZeroInputKeepsOnlyFinestUnit(t *testing.T) {
	out, err := runForTest([]string{"0"}, newTestOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "0\n"
	if out != want {
		t.Errorf("a zero quantity should show just its finest unit, got %q, want %q", out, want)
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
