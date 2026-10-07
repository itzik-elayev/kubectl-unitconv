// Package result defines the structured conversion-result model shared by
// every renderer (table, plain, json), so new output modes never need to
// re-derive conversion context from scratch.
package result

import (
	"strings"

	"github.com/itzik-elayev/kubectl-unitconv/pkg/convert"
)

// ResourceType names what kind of quantity this is, independent of which
// unit table was used to convert it: storage is memory-shaped (same units)
// but is a distinct, meaningful resource type to a reader or JSON consumer.
type ResourceType string

const (
	ResourceTypeCPU     ResourceType = "cpu"
	ResourceTypeMemory  ResourceType = "memory"
	ResourceTypeStorage ResourceType = "storage"
)

// Family returns the unit table this resource type converts through.
// Storage shares memory's unit table (Ki/Mi/Gi, bytes, ...).
func (rt ResourceType) Family() convert.Family {
	if rt == ResourceTypeCPU {
		return convert.FamilyCPU
	}
	return convert.FamilyMemory
}

// ResourceTypeFor derives a ResourceType from a resolved unit family and,
// when known, the k8s field name the quantity came from (which distinguishes
// storage from plain memory even though both use the same unit table).
func ResourceTypeFor(family convert.Family, fieldHint string) ResourceType {
	switch strings.ToLower(fieldHint) {
	case "storage", "ephemeral-storage":
		return ResourceTypeStorage
	}
	if family == convert.FamilyCPU {
		return ResourceTypeCPU
	}
	return ResourceTypeMemory
}

// Result is one converted quantity, carrying enough context to render it
// standalone: which resource it came from (when applicable), what kind of
// quantity it is, and its original and converted representations.
//
// TargetUnit is the resolved apimachinery suffix (e.g. "Gi", "m"); it is
// empty for each family's base unit (bytes for memory/storage, cores for
// cpu) — use ResourceType to tell those apart. Original and Converted are
// always strings, never JSON numbers: large quantities (e.g. byte counts
// near int64's range) lose precision in a JavaScript double, so numeric
// values are represented as exact text. Converted already includes
// TargetUnit's suffix (e.g. "1.5Gi"), matching how resource.Quantity itself
// is always written.
type Result struct {
	Kind            string       `json:"kind,omitempty"`
	Name            string       `json:"name,omitempty"`
	Namespace       string       `json:"namespace,omitempty"`
	Container       string       `json:"container,omitempty"`
	IsInitContainer bool         `json:"isInitContainer,omitempty"`
	ResourceType    ResourceType `json:"resourceType"`
	Requirement     string       `json:"requirement,omitempty"`
	Original        string       `json:"original"`
	TargetUnit      string       `json:"targetUnit"`
	Converted       string       `json:"converted"`
}
