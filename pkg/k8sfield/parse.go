package k8sfield

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// PathSegment is one step of a dotted field path, optionally indexing into
// a list (e.g. "containers[0]" -> Field: "containers", Index: &0).
type PathSegment struct {
	Field string
	Index *int
}

// FromSpec is a parsed --from argument. FieldPath is nil when the user gave
// only "TYPE/NAME" — the caller should then resolve a default path for the
// resource's kind instead of walking an explicit one.
type FromSpec struct {
	ResourceArg string
	FieldPath   []PathSegment
}

var segmentPattern = regexp.MustCompile(`^([a-zA-Z0-9_-]+)(?:\[(\d+)\])?$`)

// ParseFromSpec splits a --from argument of the form "TYPE/NAME" or
// "TYPE/NAME:field.path" into its resource reference and optional field path.
func ParseFromSpec(s string) (FromSpec, error) {
	if s == "" {
		return FromSpec{}, fmt.Errorf("--from value must not be empty")
	}

	resourceArg, rawPath, hasPath := strings.Cut(s, ":")
	if resourceArg == "" {
		return FromSpec{}, fmt.Errorf("--from value %q is missing a resource reference", s)
	}

	if !hasPath {
		return FromSpec{ResourceArg: resourceArg}, nil
	}

	fieldPath, err := ParseFieldPath(rawPath)
	if err != nil {
		return FromSpec{}, fmt.Errorf("--from value %q: %w", s, err)
	}

	return FromSpec{ResourceArg: resourceArg, FieldPath: fieldPath}, nil
}

// ParseFieldPath parses a dotted field path such as
// "spec.containers[0].resources.requests.memory" into its segments.
func ParseFieldPath(path string) ([]PathSegment, error) {
	if path == "" {
		return nil, fmt.Errorf("field path must not be empty")
	}

	rawSegments := strings.Split(path, ".")
	segments := make([]PathSegment, 0, len(rawSegments))

	for _, raw := range rawSegments {
		match := segmentPattern.FindStringSubmatch(raw)
		if match == nil {
			return nil, fmt.Errorf("invalid field path segment %q in %q", raw, path)
		}

		segment := PathSegment{Field: match[1]}
		if match[2] != "" {
			index, err := strconv.Atoi(match[2])
			if err != nil {
				return nil, fmt.Errorf("invalid index in segment %q: %w", raw, err)
			}
			segment.Index = &index
		}

		segments = append(segments, segment)
	}

	return segments, nil
}
