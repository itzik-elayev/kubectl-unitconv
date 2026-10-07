package result

import (
	"testing"

	"github.com/itzik-elayev/kubectl-unitconv/pkg/convert"
)

func TestResourceTypeFor(t *testing.T) {
	cases := []struct {
		name      string
		family    convert.Family
		fieldHint string
		want      ResourceType
	}{
		{name: "cpu family, no hint", family: convert.FamilyCPU, want: ResourceTypeCPU},
		{name: "memory family, no hint", family: convert.FamilyMemory, want: ResourceTypeMemory},
		{name: "storage field hint overrides memory family", family: convert.FamilyMemory, fieldHint: "storage", want: ResourceTypeStorage},
		{name: "ephemeral-storage field hint", family: convert.FamilyMemory, fieldHint: "ephemeral-storage", want: ResourceTypeStorage},
		{name: "memory field hint stays memory", family: convert.FamilyMemory, fieldHint: "memory", want: ResourceTypeMemory},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ResourceTypeFor(c.family, c.fieldHint)
			if got != c.want {
				t.Errorf("ResourceTypeFor(%v, %q) = %q, want %q", c.family, c.fieldHint, got, c.want)
			}
		})
	}
}

func TestResourceTypeFamily(t *testing.T) {
	if ResourceTypeCPU.Family() != convert.FamilyCPU {
		t.Error("cpu resource type must map to the cpu unit family")
	}
	if ResourceTypeMemory.Family() != convert.FamilyMemory {
		t.Error("memory resource type must map to the memory unit family")
	}
	if ResourceTypeStorage.Family() != convert.FamilyMemory {
		t.Error("storage resource type must map to the memory unit family (same unit table)")
	}
}
