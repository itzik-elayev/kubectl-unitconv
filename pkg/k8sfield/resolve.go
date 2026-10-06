package k8sfield

import "fmt"

// WalkFieldPath descends an unstructured object (as decoded from JSON: nested
// map[string]any and []any) following segments, and returns
// the leaf as a string — resource.Quantity fields always marshal as JSON
// strings in real k8s API types.
func WalkFieldPath(obj map[string]any, segments []PathSegment) (string, error) {
	if len(segments) == 0 {
		return "", fmt.Errorf("field path must not be empty")
	}

	var current any = obj

	for i, segment := range segments {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return "", fmt.Errorf("cannot descend into field %q: not an object", segment.Field)
		}

		next, ok := currentMap[segment.Field]
		if !ok {
			return "", fmt.Errorf("field %q not found", segment.Field)
		}

		if segment.Index != nil {
			list, ok := next.([]any)
			if !ok {
				return "", fmt.Errorf("field %q is not a list, cannot index into it", segment.Field)
			}
			if *segment.Index < 0 || *segment.Index >= len(list) {
				return "", fmt.Errorf("index %d out of range for field %q (len %d)", *segment.Index, segment.Field, len(list))
			}
			next = list[*segment.Index]
		}

		if i == len(segments)-1 {
			leaf, ok := next.(string)
			if !ok {
				return "", fmt.Errorf("field %q is not a string value", segment.Field)
			}
			return leaf, nil
		}

		current = next
	}

	return "", fmt.Errorf("field path resolved to no value")
}
