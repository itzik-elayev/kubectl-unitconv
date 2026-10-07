package k8sfield

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/itzik-elayev/kubectl-unitconv/pkg/convert"
)

func newContainer(name string, requests, limits map[string]string) map[string]any {
	resources := map[string]any{}
	if requests != nil {
		resources["requests"] = stringMapToAny(requests)
	}
	if limits != nil {
		resources["limits"] = stringMapToAny(limits)
	}
	return map[string]any{"name": name, "resources": resources}
}

func stringMapToAny(m map[string]string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func testPod() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"kind": "Pod",
		"spec": map[string]any{
			"containers": []any{
				newContainer("app", map[string]string{"memory": "500Mi", "cpu": "250m"}, map[string]string{"memory": "1Gi"}),
				newContainer("sidecar", map[string]string{"memory": "64Mi"}, nil),
			},
			"initContainers": []any{
				newContainer("init-setup", map[string]string{"memory": "32Mi"}, map[string]string{"memory": "64Mi"}),
			},
		},
	}}
}

func readingKey(r Reading) string {
	key := r.Container + "/" + r.Requirement
	if r.IsInit {
		key = "init:" + key
	}
	return key
}

func TestResolveDefaultPodFansOutByDefault(t *testing.T) {
	readings, err := ResolveDefault(testPod(), FromOptions{}, convert.FamilyMemory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]string{
		"app/requests":             "500Mi",
		"app/limits":               "1Gi",
		"sidecar/requests":         "64Mi",
		"init:init-setup/requests": "32Mi",
		"init:init-setup/limits":   "64Mi",
	}

	if len(readings) != len(want) {
		t.Fatalf("got %d readings, want %d: %+v", len(readings), len(want), readings)
	}
	for _, r := range readings {
		wantValue, ok := want[readingKey(r)]
		if !ok {
			t.Errorf("unexpected reading %+v", r)
			continue
		}
		if r.RawValue != wantValue {
			t.Errorf("reading %s = %q, want %q", readingKey(r), r.RawValue, wantValue)
		}
	}
}

func TestResolveDefaultPodFiltersByContainer(t *testing.T) {
	readings, err := ResolveDefault(testPod(), FromOptions{Container: "sidecar"}, convert.FamilyMemory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(readings) != 1 || readings[0].RawValue != "64Mi" {
		t.Fatalf("got %+v, want single sidecar requests reading", readings)
	}
}

func TestResolveDefaultPodContainerNotFound(t *testing.T) {
	_, err := ResolveDefault(testPod(), FromOptions{Container: "nonexistent"}, convert.FamilyMemory)
	if err == nil {
		t.Fatal("expected error for unknown container name")
	}
}

func TestResolveDefaultPodFiltersByRequirement(t *testing.T) {
	readings, err := ResolveDefault(testPod(), FromOptions{Requirement: "limits"}, convert.FamilyMemory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]string{"app/limits": "1Gi", "init:init-setup/limits": "64Mi"}
	if len(readings) != len(want) {
		t.Fatalf("got %d readings, want %d: %+v", len(readings), len(want), readings)
	}
	for _, r := range readings {
		if r.Requirement != "limits" {
			t.Errorf("reading %+v has requirement %q, want limits", r, r.Requirement)
		}
	}
}

func TestResolveDefaultPodCPUFamily(t *testing.T) {
	readings, err := ResolveDefault(testPod(), FromOptions{}, convert.FamilyCPU)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(readings) != 1 || readings[0].Container != "app" || readings[0].RawValue != "250m" {
		t.Fatalf("got %+v, want single app requests cpu reading", readings)
	}
}

func TestResolveDefaultPodAllMissingErrors(t *testing.T) {
	_, err := ResolveDefault(testPod(), FromOptions{Container: "sidecar"}, convert.FamilyCPU)
	if err == nil {
		t.Fatal("expected error when no reading matches family/filters")
	}
}

func TestResolveDefaultDeploymentUsesTemplatePath(t *testing.T) {
	deployment := &unstructured.Unstructured{Object: map[string]any{
		"kind": "Deployment",
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []any{
						newContainer("app", map[string]string{"memory": "2Gi"}, nil),
					},
				},
			},
		},
	}}

	readings, err := ResolveDefault(deployment, FromOptions{}, convert.FamilyMemory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(readings) != 1 || readings[0].RawValue != "2Gi" {
		t.Fatalf("got %+v, want single app requests reading", readings)
	}
}

func TestResolveDefaultStatefulSetUsesTemplatePath(t *testing.T) {
	statefulSet := &unstructured.Unstructured{Object: map[string]any{
		"kind": "StatefulSet",
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []any{
						newContainer("app", map[string]string{"memory": "3Gi"}, nil),
					},
				},
			},
		},
	}}

	readings, err := ResolveDefault(statefulSet, FromOptions{}, convert.FamilyMemory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(readings) != 1 || readings[0].RawValue != "3Gi" {
		t.Fatalf("got %+v, want single app requests reading", readings)
	}
}

func TestResolveDefaultPVC(t *testing.T) {
	pvc := &unstructured.Unstructured{Object: map[string]any{
		"kind": "PersistentVolumeClaim",
		"spec": map[string]any{
			"resources": map[string]any{
				"requests": map[string]any{"storage": "10Gi"},
			},
		},
	}}

	readings, err := ResolveDefault(pvc, FromOptions{}, convert.FamilyMemory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(readings) != 1 || readings[0].RawValue != "10Gi" {
		t.Fatalf("got %+v, want single storage reading", readings)
	}
}

func TestResolveDefaultPVCRejectsCPUFamily(t *testing.T) {
	pvc := &unstructured.Unstructured{Object: map[string]any{
		"kind": "PersistentVolumeClaim",
		"spec": map[string]any{
			"resources": map[string]any{
				"requests": map[string]any{"storage": "10Gi"},
			},
		},
	}}

	_, err := ResolveDefault(pvc, FromOptions{}, convert.FamilyCPU)
	if err == nil {
		t.Fatal("expected error treating PVC storage as a cpu quantity")
	}
}

func TestResolveDefaultNode(t *testing.T) {
	node := &unstructured.Unstructured{Object: map[string]any{
		"kind": "Node",
		"status": map[string]any{
			"allocatable": map[string]any{"memory": "16Gi", "cpu": "4"},
		},
	}}

	memory, err := ResolveDefault(node, FromOptions{}, convert.FamilyMemory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(memory) != 1 || memory[0].RawValue != "16Gi" {
		t.Fatalf("got %+v, want single memory reading", memory)
	}

	cpu, err := ResolveDefault(node, FromOptions{}, convert.FamilyCPU)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cpu) != 1 || cpu[0].RawValue != "4" {
		t.Fatalf("got %+v, want single cpu reading", cpu)
	}
}

func TestResolveDefaultUnknownKind(t *testing.T) {
	configMap := &unstructured.Unstructured{Object: map[string]any{"kind": "ConfigMap"}}

	_, err := ResolveDefault(configMap, FromOptions{}, convert.FamilyMemory)
	if err == nil {
		t.Fatal("expected error for a kind with no default resolver")
	}
}
