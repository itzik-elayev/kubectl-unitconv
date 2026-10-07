package render

import (
	"encoding/json"

	"github.com/itzik-elayev/kubectl-unitconv/pkg/result"
)

// JSON renders results as an indented JSON array, always valid and free of
// ANSI codes or decorative text — safe for machine consumption regardless
// of the requested color mode.
func JSON(results []result.Result) (string, error) {
	if results == nil {
		results = []result.Result{}
	}
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
