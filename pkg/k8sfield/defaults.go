package k8sfield

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/itzik-elayev/kubectl-unitconv/pkg/convert"
)

// FromOptions narrows a default field-path lookup. Both fields default to
// "no filter": every container (regular + init) and both requirements.
type FromOptions struct {
	Container   string
	Requirement string
}

// Reading is one resolved quantity. Container/Requirement are "" for kinds
// without that concept (pvc, node) — there's always exactly one Reading then.
type Reading struct {
	Container   string
	IsInit      bool
	Requirement string
	RawValue    string
}

// DefaultResolver reads every "obvious" quantity off a known kind directly
// via unstructured.Nested* helpers — no PathSegment indirection needed since
// each kind's shape is known statically.
type DefaultResolver func(obj *unstructured.Unstructured, opts FromOptions, family convert.Family) ([]Reading, error)

var defaultResolvers = map[string]DefaultResolver{
	"pod":                   containerQuantityResolver([]string{"spec", "containers"}, []string{"spec", "initContainers"}),
	"deployment":            containerQuantityResolver([]string{"spec", "template", "spec", "containers"}, []string{"spec", "template", "spec", "initContainers"}),
	"statefulset":           containerQuantityResolver([]string{"spec", "template", "spec", "containers"}, []string{"spec", "template", "spec", "initContainers"}),
	"daemonset":             containerQuantityResolver([]string{"spec", "template", "spec", "containers"}, []string{"spec", "template", "spec", "initContainers"}),
	"replicaset":            containerQuantityResolver([]string{"spec", "template", "spec", "containers"}, []string{"spec", "template", "spec", "initContainers"}),
	"job":                   containerQuantityResolver([]string{"spec", "template", "spec", "containers"}, []string{"spec", "template", "spec", "initContainers"}),
	"persistentvolumeclaim": pvcStorageResolver,
	"node":                  nodeAllocatableResolver,
}

// ResolveDefault resolves every matching quantity for obj's own kind,
// without needing an explicit --from field path.
func ResolveDefault(obj *unstructured.Unstructured, opts FromOptions, family convert.Family) ([]Reading, error) {
	resolver, ok := defaultResolvers[strings.ToLower(obj.GetKind())]
	if !ok {
		return nil, fmt.Errorf("no default field path for kind %q — specify --from %s/%s:field.path", obj.GetKind(), obj.GetKind(), obj.GetName())
	}
	return resolver(obj, opts, family)
}

func familyKey(family convert.Family) string {
	if family == convert.FamilyCPU {
		return "cpu"
	}
	return "memory"
}

// containerQuantityResolver builds a DefaultResolver for any kind whose pod
// spec lives at containersPath/initContainersPath — pods have it directly,
// workload kinds (deployment, statefulset, ...) nest it under spec.template.
func containerQuantityResolver(containersPath, initContainersPath []string) DefaultResolver {
	return func(obj *unstructured.Unstructured, opts FromOptions, family convert.Family) ([]Reading, error) {
		containers, err := namedContainers(obj, containersPath, false)
		if err != nil {
			return nil, err
		}
		initContainers, err := namedContainers(obj, initContainersPath, true)
		if err != nil {
			return nil, err
		}
		allContainers := append(containers, initContainers...)

		if opts.Container != "" {
			allContainers = filterByName(allContainers, opts.Container)
			if len(allContainers) == 0 {
				return nil, fmt.Errorf("no container named %q found on %s/%s", opts.Container, obj.GetKind(), obj.GetName())
			}
		}

		requirements := []string{"requests", "limits"}
		if opts.Requirement != "" {
			requirements = []string{opts.Requirement}
		}

		var readings []Reading
		for _, c := range allContainers {
			for _, requirement := range requirements {
				value, found, err := unstructured.NestedString(c.fields, "resources", requirement, familyKey(family))
				if err != nil {
					return nil, fmt.Errorf("container %q: %w", c.name, err)
				}
				if !found || value == "" {
					continue
				}
				readings = append(readings, Reading{
					Container:   c.name,
					IsInit:      c.isInit,
					Requirement: requirement,
					RawValue:    value,
				})
			}
		}

		if len(readings) == 0 {
			return nil, fmt.Errorf("no %s quantity found in requests/limits on any container of %s/%s", familyKey(family), obj.GetKind(), obj.GetName())
		}
		return readings, nil
	}
}

type namedContainer struct {
	name   string
	isInit bool
	fields map[string]any
}

func namedContainers(obj *unstructured.Unstructured, path []string, isInit bool) ([]namedContainer, error) {
	raw, found, err := unstructured.NestedSlice(obj.Object, path...)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", strings.Join(path, "."), err)
	}
	if !found {
		return nil, nil
	}

	containers := make([]namedContainer, 0, len(raw))
	for _, item := range raw {
		fields, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _, _ := unstructured.NestedString(fields, "name")
		containers = append(containers, namedContainer{name: name, isInit: isInit, fields: fields})
	}
	return containers, nil
}

func filterByName(containers []namedContainer, name string) []namedContainer {
	filtered := make([]namedContainer, 0, 1)
	for _, c := range containers {
		if c.name == name {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// pvcStorageResolver ignores opts/family — a PVC has one sensible default
// quantity. status.capacity.storage or other variants need an explicit path.
func pvcStorageResolver(obj *unstructured.Unstructured, _ FromOptions, _ convert.Family) ([]Reading, error) {
	value, found, err := unstructured.NestedString(obj.Object, "spec", "resources", "requests", "storage")
	if err != nil {
		return nil, fmt.Errorf("reading spec.resources.requests.storage: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("no spec.resources.requests.storage set on %s/%s", obj.GetKind(), obj.GetName())
	}
	return []Reading{{RawValue: value}}, nil
}

// nodeAllocatableResolver reads status.allocatable.<family>. status.capacity.*
// needs an explicit path.
func nodeAllocatableResolver(obj *unstructured.Unstructured, _ FromOptions, family convert.Family) ([]Reading, error) {
	value, found, err := unstructured.NestedString(obj.Object, "status", "allocatable", familyKey(family))
	if err != nil {
		return nil, fmt.Errorf("reading status.allocatable.%s: %w", familyKey(family), err)
	}
	if !found {
		return nil, fmt.Errorf("no status.allocatable.%s set on %s/%s", familyKey(family), obj.GetKind(), obj.GetName())
	}
	return []Reading{{RawValue: value}}, nil
}
