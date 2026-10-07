package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/itzik-elayev/kubectl-unitconv/pkg/result"
)

// These check structure (root label, tree connectors, final branch), not a
// snapshot of the fully decorated/colored table — brittle to style tweaks.

func TestTableShowAllRendersAsTree(t *testing.T) {
	var buf bytes.Buffer
	Table(&buf, []result.Result{
		{ResourceType: result.ResourceTypeMemory, Original: "500Mi", TargetUnit: "Mi", Converted: "500Mi"},
		{ResourceType: result.ResourceTypeMemory, Original: "500Mi", TargetUnit: "Gi", Converted: "0.488281Gi"},
	}, false, 0)

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	if lines[0] != "500Mi" {
		t.Errorf("root line = %q, want the original value %q", lines[0], "500Mi")
	}
	if !strings.HasPrefix(lines[1], "├─ ") {
		t.Errorf("first branch should use the mid-tree connector, got %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "└─ ") {
		t.Errorf("last branch should use the terminal connector, got %q", lines[2])
	}
	if !strings.Contains(lines[2], "0.488281Gi") {
		t.Errorf("expected converted value in the last branch, got %q", lines[2])
	}
}

func TestTableShowAllRootIncludesReadingContext(t *testing.T) {
	var buf bytes.Buffer
	Table(&buf, []result.Result{
		{Kind: "Pod", Name: "my-app", Container: "app", ResourceType: result.ResourceTypeMemory, Requirement: "requests", Original: "500Mi", TargetUnit: "Mi", Converted: "500Mi"},
		{Kind: "Pod", Name: "my-app", Container: "app", ResourceType: result.ResourceTypeMemory, Requirement: "requests", Original: "500Mi", TargetUnit: "Gi", Converted: "0.488281Gi"},
	}, false, 0)

	root, _, _ := strings.Cut(buf.String(), "\n")
	for _, want := range []string{"Pod/my-app", "container app (requests)", "500Mi"} {
		if !strings.Contains(root, want) {
			t.Errorf("root line %q should contain %q", root, want)
		}
	}
}

func TestTableCompactConversionIsNotATree(t *testing.T) {
	var buf bytes.Buffer
	Table(&buf, []result.Result{
		{ResourceType: result.ResourceTypeMemory, Original: "500Mi", TargetUnit: "Gi", Converted: "0.488281Gi"},
	}, false, 0)

	out := buf.String()
	if strings.Contains(out, "├─") || strings.Contains(out, "└─") {
		t.Errorf("a single explicit-target conversion should render compact, not as a tree: %q", out)
	}
	if !strings.Contains(out, "500Mi (0.488281Gi)") {
		t.Errorf("expected compact \"original (converted)\" form, got %q", out)
	}
}
