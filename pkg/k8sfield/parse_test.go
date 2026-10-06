package k8sfield

import (
	"reflect"
	"testing"
)

func TestParseFromSpec(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    FromSpec
		wantErr bool
	}{
		{
			name:  "resource only, no path",
			input: "pod/foo",
			want:  FromSpec{ResourceArg: "pod/foo", FieldPath: nil},
		},
		{
			name:  "resource with explicit path",
			input: "pod/foo:spec.containers[0].resources.requests.memory",
			want: FromSpec{
				ResourceArg: "pod/foo",
				FieldPath: []PathSegment{
					{Field: "spec"},
					{Field: "containers", Index: new(int)},
					{Field: "resources"},
					{Field: "requests"},
					{Field: "memory"},
				},
			},
		},
		{name: "empty input", input: "", wantErr: true},
		{name: "empty path after colon", input: "pod/foo:", wantErr: true},
		{name: "malformed path segment", input: "pod/foo:a:b", wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseFromSpec(c.input)

			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("ParseFromSpec(%q) = %+v, want %+v", c.input, got, c.want)
			}
		})
	}
}

func TestParseFieldPath(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    []PathSegment
		wantErr bool
	}{
		{
			name:  "simple dotted path",
			input: "status.allocatable.memory",
			want: []PathSegment{
				{Field: "status"},
				{Field: "allocatable"},
				{Field: "memory"},
			},
		},
		{
			name:  "with index",
			input: "spec.containers[0].resources.limits.cpu",
			want: []PathSegment{
				{Field: "spec"},
				{Field: "containers", Index: new(int)},
				{Field: "resources"},
				{Field: "limits"},
				{Field: "cpu"},
			},
		},
		{name: "empty", input: "", wantErr: true},
		{name: "invalid characters", input: "spec.contain ers", wantErr: true},
		{name: "unclosed bracket", input: "spec.containers[0", wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseFieldPath(c.input)

			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("ParseFieldPath(%q) = %+v, want %+v", c.input, got, c.want)
			}
		})
	}
}
