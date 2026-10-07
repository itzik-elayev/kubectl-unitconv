package render

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/itzik-elayev/kubectl-unitconv/pkg/result"
)

func TestJSONIsValidAndFreeOfANSI(t *testing.T) {
	results := []result.Result{
		{Kind: "Pod", Name: "app", Namespace: "default", Container: "app", ResourceType: result.ResourceTypeMemory, Requirement: "requests", Original: "500Mi", TargetUnit: "Gi", Converted: "0.488281Gi"},
		{ResourceType: result.ResourceTypeCPU, Original: "250m", TargetUnit: "", Converted: "0.25"},
	}

	out, err := JSON(results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded []result.Result
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("JSON output is not valid JSON: %v\n%s", err, out)
	}
	if len(decoded) != len(results) {
		t.Fatalf("got %d decoded results, want %d", len(decoded), len(results))
	}

	const ansiEscape = "\x1b["
	if strings.Contains(out, ansiEscape) {
		t.Error("JSON output must never contain ANSI escape codes")
	}
}

func TestJSONEmptyResultsIsAnEmptyArray(t *testing.T) {
	out, err := JSON(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("got %q, want an empty JSON array", out)
	}
}

func TestJSONNumericValuesAreStrings(t *testing.T) {
	// A quantity near int64's range would lose precision as a JSON number
	// in a JS consumer; original/converted must always be JSON strings.
	results := []result.Result{
		{ResourceType: result.ResourceTypeMemory, Original: "9223372036854775807", TargetUnit: "", Converted: "9223372036854775807"},
	}

	out, err := JSON(results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var raw []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, field := range []string{"original", "converted"} {
		value := string(raw[0][field])
		if !strings.HasPrefix(value, `"`) {
			t.Errorf("field %q = %s, want a JSON string (quoted), not a bare number", field, value)
		}
	}
}

func TestJSONPreservesOriginalAcrossShowAllExpansion(t *testing.T) {
	results := []result.Result{
		{ResourceType: result.ResourceTypeMemory, Original: "500Mi", TargetUnit: "", Converted: "524288000"},
		{ResourceType: result.ResourceTypeMemory, Original: "500Mi", TargetUnit: "Gi", Converted: "0.488281Gi"},
	}

	for _, r := range results {
		if r.Original != "500Mi" {
			t.Errorf("Original = %q, want the exact original input preserved across every expanded row", r.Original)
		}
	}
}
