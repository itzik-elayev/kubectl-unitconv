package k8sfield

import "testing"

func TestWalkFieldPath(t *testing.T) {
	obj := map[string]any{
		"spec": map[string]any{
			"containers": []any{
				map[string]any{
					"name": "app",
					"resources": map[string]any{
						"requests": map[string]any{
							"memory": "500Mi",
						},
					},
				},
				map[string]any{
					"name": "sidecar",
					"resources": map[string]any{
						"requests": map[string]any{
							"memory": "64Mi",
						},
					},
				},
			},
		},
		"status": map[string]any{
			"allocatable": map[string]any{
				"memory": "16Gi",
			},
		},
	}

	cases := []struct {
		name     string
		segments []PathSegment
		want     string
		wantErr  bool
	}{
		{
			name: "indexed container lookup",
			segments: []PathSegment{
				{Field: "spec"}, {Field: "containers", Index: new(int)},
				{Field: "resources"}, {Field: "requests"}, {Field: "memory"},
			},
			want: "500Mi",
		},
		{
			name: "second container",
			segments: []PathSegment{
				{Field: "spec"}, {Field: "containers", Index: intPtr(1)},
				{Field: "resources"}, {Field: "requests"}, {Field: "memory"},
			},
			want: "64Mi",
		},
		{
			name:     "no index needed",
			segments: []PathSegment{{Field: "status"}, {Field: "allocatable"}, {Field: "memory"}},
			want:     "16Gi",
		},
		{
			name:     "missing field",
			segments: []PathSegment{{Field: "spec"}, {Field: "nope"}},
			wantErr:  true,
		},
		{
			name: "index out of range",
			segments: []PathSegment{
				{Field: "spec"}, {Field: "containers", Index: intPtr(5)},
			},
			wantErr: true,
		},
		{
			name:     "index on non-list field",
			segments: []PathSegment{{Field: "status", Index: new(int)}},
			wantErr:  true,
		},
		{
			name:     "leaf is not a string",
			segments: []PathSegment{{Field: "spec"}, {Field: "containers"}},
			wantErr:  true,
		},
		{
			name:     "empty path",
			segments: nil,
			wantErr:  true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := WalkFieldPath(obj, c.segments)

			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("WalkFieldPath(...) = %q, want %q", got, c.want)
			}
		})
	}
}

func intPtr(i int) *int { return &i }
