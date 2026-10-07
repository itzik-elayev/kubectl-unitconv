package render

import (
	"bytes"
	"testing"

	"github.com/itzik-elayev/kubectl-unitconv/pkg/result"
)

// These assert the plain renderer reproduces kubectl-unitconv's original,
// pre-table text output exactly — the contract "plain preserves existing
// output behavior" depends on this byte-for-byte.

func TestPlainLiteralSingleTarget(t *testing.T) {
	var buf bytes.Buffer
	Plain(&buf, []result.Result{
		{ResourceType: result.ResourceTypeMemory, Original: "500Mi", TargetUnit: "Gi", Converted: "0.488281Gi"},
	})

	want := "0.488281Gi\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestPlainLiteralShowAll(t *testing.T) {
	var buf bytes.Buffer
	Plain(&buf, []result.Result{
		{ResourceType: result.ResourceTypeCPU, Original: "250m", TargetUnit: "n", Converted: "250000000n"},
		{ResourceType: result.ResourceTypeCPU, Original: "250m", TargetUnit: "m", Converted: "250m"},
		{ResourceType: result.ResourceTypeCPU, Original: "250m", TargetUnit: "", Converted: "0.25"},
	})

	want := "nanocores            250000000n\n" +
		"millicores           250m\n" +
		"cores                0.25\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestPlainFromReadingSingleTarget(t *testing.T) {
	var buf bytes.Buffer
	Plain(&buf, []result.Result{
		{Kind: "Pod", Name: "x", Container: "app", ResourceType: result.ResourceTypeMemory, Requirement: "requests", Original: "500Mi", TargetUnit: "Gi", Converted: "0.488281Gi"},
	})

	want := "container app (requests): 0.488281Gi\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestPlainFromReadingInitContainerNoRequirementLabel(t *testing.T) {
	var buf bytes.Buffer
	Plain(&buf, []result.Result{
		{Kind: "Pod", Name: "x", Container: "init-setup", IsInitContainer: true, ResourceType: result.ResourceTypeMemory, Original: "32Mi", TargetUnit: "Gi", Converted: "0.03125Gi"},
	})

	want := "init container init-setup: 0.03125Gi\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestPlainFromReadingShowAllIndentsUnderLabel(t *testing.T) {
	var buf bytes.Buffer
	Plain(&buf, []result.Result{
		{Kind: "Pod", Name: "x", Container: "app", ResourceType: result.ResourceTypeMemory, Requirement: "requests", Original: "500Mi", TargetUnit: "", Converted: "524288000"},
		{Kind: "Pod", Name: "x", Container: "app", ResourceType: result.ResourceTypeMemory, Requirement: "requests", Original: "500Mi", TargetUnit: "Gi", Converted: "0.488281Gi"},
	})

	want := "container app (requests):\n" +
		"  bytes                524288000\n" +
		"  gibibytes (2^30)     0.488281Gi\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestPlainFanOutPrintsOneLinePerGroupInOrder(t *testing.T) {
	var buf bytes.Buffer
	Plain(&buf, []result.Result{
		{Kind: "Pod", Name: "x", Container: "app", ResourceType: result.ResourceTypeMemory, Requirement: "requests", Original: "500Mi", TargetUnit: "Gi", Converted: "0.488281Gi"},
		{Kind: "Pod", Name: "x", Container: "sidecar", ResourceType: result.ResourceTypeMemory, Requirement: "requests", Original: "64Mi", TargetUnit: "Gi", Converted: "0.0625Gi"},
	})

	want := "container app (requests): 0.488281Gi\n" +
		"container sidecar (requests): 0.0625Gi\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}
