package reporting

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/magnusbaeck/logstash-filter-verifier/v2/internal/comparison"
)

func sampleReport() RunReport {
	return RunReport{
		StartedAt: time.Now().UTC(),
		Suites: []SuiteReport{
			{
				Name: "syslog.yaml",
				TestCases: []TestCaseReport{
					{Name: "valid message", Status: StatusPassed, EventIndex: 0},
					{
						Name: "invalid pid", Status: StatusFailed, EventIndex: 1,
						Diffs: []comparison.FieldDiff{{
							Kind: comparison.TypeMismatch, Path: "$.pid",
							Expected: "31993", Actual: float64(31993),
						}},
					},
				},
			},
		},
	}
}

func TestRenderJSON(t *testing.T) {
	body, err := RenderJSON(sampleReport())
	if err != nil {
		t.Fatal(err)
	}

	var report RunReport
	if err := json.Unmarshal(body, &report); err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusFailed {
		t.Fatalf("expected failed report, got %q", report.Status)
	}
	if report.Summary.TestCases != 2 || report.Summary.Passed != 1 || report.Summary.Failed != 1 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
}

func TestRenderJUnit(t *testing.T) {
	body, err := RenderJUnit(sampleReport())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(body), xml.Header) {
		t.Fatal("JUnit report must start with an XML header")
	}
	if !strings.Contains(string(body), `<testsuites name="logstash-filter-verifier"`) {
		t.Fatalf("missing testsuites element:\n%s", body)
	}
	if !strings.Contains(string(body), `<failure message="Logstash event comparison failed">`) {
		t.Fatalf("missing failure element:\n%s", body)
	}
}
