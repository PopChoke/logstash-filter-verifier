package comparison

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type DiffKind string

const (
	MissingField       DiffKind = "missing_field"
	UnexpectedField    DiffKind = "unexpected_field"
	ValueMismatch      DiffKind = "value_mismatch"
	TypeMismatch       DiffKind = "type_mismatch"
	LengthMismatch     DiffKind = "length_mismatch"
	EventCountMismatch DiffKind = "event_count_mismatch"
)

type FieldDiff struct {
	Kind     DiffKind    `json:"kind"`
	Path     string      `json:"path"`
	Expected interface{} `json:"expected,omitempty"`
	Actual   interface{} `json:"actual,omitempty"`
}

func Compare(expected, actual map[string]interface{}) []FieldDiff {
	diffs := make([]FieldDiff, 0)
	compareObjects("$", expected, actual, &diffs)
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Path < diffs[j].Path
	})
	return diffs
}

func compareObjects(path string, expected, actual map[string]interface{}, diffs *[]FieldDiff) {
	keys := make(map[string]struct{}, len(expected)+len(actual))
	for key := range expected {
		keys[key] = struct{}{}
	}
	for key := range actual {
		keys[key] = struct{}{}
	}

	sortedKeys := make([]string, 0, len(keys))
	for key := range keys {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	for _, key := range sortedKeys {
		expectedValue, expectedExists := expected[key]
		actualValue, actualExists := actual[key]
		fieldPath := joinObjectPath(path, key)

		if !expectedExists {
			*diffs = append(*diffs, FieldDiff{
				Kind: UnexpectedField, Path: fieldPath, Actual: actualValue,
			})
			continue
		}
		if !actualExists {
			*diffs = append(*diffs, FieldDiff{
				Kind: MissingField, Path: fieldPath, Expected: expectedValue,
			})
			continue
		}
		compareValues(fieldPath, expectedValue, actualValue, diffs)
	}
}

func compareValues(path string, expected, actual interface{}, diffs *[]FieldDiff) {
	if reflect.TypeOf(expected) != reflect.TypeOf(actual) {
		*diffs = append(*diffs, FieldDiff{
			Kind: TypeMismatch, Path: path, Expected: expected, Actual: actual,
		})
		return
	}

	switch expectedValue := expected.(type) {
	case map[string]interface{}:
		actualValue, ok := actual.(map[string]interface{})
		if ok {
			compareObjects(path, expectedValue, actualValue, diffs)
		}
	case []interface{}:
		actualValue, ok := actual.([]interface{})
		if !ok {
			*diffs = append(*diffs, FieldDiff{
				Kind: ValueMismatch, Path: path, Expected: expected, Actual: actual,
			})
			return
		}
		if len(expectedValue) != len(actualValue) {
			*diffs = append(*diffs, FieldDiff{
				Kind: LengthMismatch, Path: path, Expected: len(expectedValue), Actual: len(actualValue),
			})
		}
		commonLength := len(expectedValue)
		if len(actualValue) < commonLength {
			commonLength = len(actualValue)
		}
		for i := 0; i < commonLength; i++ {
			compareValues(fmt.Sprintf("%s[%d]", path, i), expectedValue[i], actualValue[i], diffs)
		}
		for i := commonLength; i < len(expectedValue); i++ {
			*diffs = append(*diffs, FieldDiff{
				Kind: MissingField, Path: fmt.Sprintf("%s[%d]", path, i), Expected: expectedValue[i],
			})
		}
		for i := commonLength; i < len(actualValue); i++ {
			*diffs = append(*diffs, FieldDiff{
				Kind: UnexpectedField, Path: fmt.Sprintf("%s[%d]", path, i), Actual: actualValue[i],
			})
		}
	default:
		if !reflect.DeepEqual(expected, actual) {
			*diffs = append(*diffs, FieldDiff{
				Kind: ValueMismatch, Path: path, Expected: expected, Actual: actual,
			})
		}
	}
}

func joinObjectPath(path, key string) string {
	if key == "" {
		return path + "."
	}
	if strings.ContainsAny(key, ".[") {
		return path + "[\"" + strings.ReplaceAll(key, "\"", "\\\"") + "\"]"
	}
	return path + "." + key
}

func Format(diffs []FieldDiff) string {
	if len(diffs) == 0 {
		return ""
	}

	lines := make([]string, 0, len(diffs))
	for _, diff := range diffs {
		switch diff.Kind {
		case MissingField:
			lines = append(lines, fmt.Sprintf("- %s: missing field (expected %v)", diff.Path, diff.Expected))
		case UnexpectedField:
			lines = append(lines, fmt.Sprintf("- %s: unexpected field (actual %v)", diff.Path, diff.Actual))
		case TypeMismatch:
			lines = append(lines, fmt.Sprintf("- %s: type mismatch (expected %T, actual %T)", diff.Path, diff.Expected, diff.Actual))
		case LengthMismatch:
			lines = append(lines, fmt.Sprintf("- %s: array length mismatch (expected %v, actual %v)", diff.Path, diff.Expected, diff.Actual))
		case EventCountMismatch:
			lines = append(lines, fmt.Sprintf("- $: event count mismatch (expected %v, actual %v)", diff.Expected, diff.Actual))
		case ValueMismatch:
			lines = append(lines, fmt.Sprintf("- %s: value mismatch (expected %v, actual %v)", diff.Path, diff.Expected, diff.Actual))
		default:
			lines = append(lines, fmt.Sprintf("- %s: %s", diff.Path, diff.Kind))
		}
	}
	return strings.Join(lines, "\n")
}
