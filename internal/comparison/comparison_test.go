package comparison

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareNestedFields(t *testing.T) {
	expected := map[string]interface{}{
		"a": map[string]interface{}{
			"pid": "31993",
		},
		"tags": []interface{}{"valid", "mail"},
	}
	actual := map[string]interface{}{
		"a": map[string]interface{}{
			"pid": float64(31993),
		},
		"tags": []interface{}{"valid"},
		"host": "runner-01",
	}

	diffs := Compare(expected, actual)

	assert.Len(t, diffs, 4)
	assert.Contains(t, Format(diffs), "$.a.pid: type mismatch")
	assert.Contains(t, Format(diffs), "$.host: unexpected field")
	assert.Contains(t, Format(diffs), "$.tags: array length mismatch")
}

func TestCompareMissingAndValueMismatch(t *testing.T) {
	diffs := Compare(
		map[string]interface{}{"a": "b", "missing": true},
		map[string]interface{}{"a": "B"},
	)

	assert.Len(t, diffs, 2)
	assert.Contains(t, Format(diffs), "$.a: value mismatch")
	assert.Contains(t, Format(diffs), "$.missing: missing field")
}

func TestCompareArrayEntries(t *testing.T) {
	diffs := Compare(
		map[string]interface{}{"items": []interface{}{"one", "two", "three"}},
		map[string]interface{}{"items": []interface{}{"one", "changed", "three", "extra"}},
	)

	formatted := Format(diffs)
	assert.Contains(t, formatted, "$.items[1]: value mismatch")
	assert.Contains(t, formatted, "$.items: array length mismatch")
	assert.Contains(t, formatted, "$.items[3]: unexpected field")
}
