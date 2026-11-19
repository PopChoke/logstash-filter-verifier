package reporting

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/magnusbaeck/logstash-filter-verifier/v2/internal/comparison"
)

type Status string

const (
	StatusPassed Status = "passed"
	StatusFailed Status = "failed"
)

type RunReport struct {
	Status     Status        `json:"status"`
	StartedAt  time.Time     `json:"started_at"`
	FinishedAt time.Time     `json:"finished_at"`
	Suites     []SuiteReport `json:"suites"`
	Summary    Summary       `json:"summary"`
}

type Summary struct {
	Suites    int `json:"suites"`
	TestCases int `json:"test_cases"`
	Passed    int `json:"passed"`
	Failed    int `json:"failed"`
}

type SuiteReport struct {
	Name      string           `json:"name"`
	Status    Status           `json:"status"`
	TestCases []TestCaseReport `json:"test_cases"`
}

type TestCaseReport struct {
	Name       string                 `json:"name"`
	Status     Status                 `json:"status"`
	EventIndex int                    `json:"event_index"`
	Explain    string                 `json:"explanation,omitempty"`
	Diffs      []comparison.FieldDiff `json:"diffs,omitempty"`
}

func (r *RunReport) Recalculate() {
	r.Summary = Summary{Suites: len(r.Suites)}
	r.Status = StatusPassed

	for i := range r.Suites {
		r.Suites[i].Status = StatusPassed
		for _, testCase := range r.Suites[i].TestCases {
			r.Summary.TestCases++
			if testCase.Status == StatusPassed {
				r.Summary.Passed++
			} else {
				r.Summary.Failed++
				r.Suites[i].Status = StatusFailed
				r.Status = StatusFailed
			}
		}
	}
	r.FinishedAt = time.Now().UTC()
}

func RenderJSON(report RunReport) ([]byte, error) {
	report.Recalculate()
	return json.MarshalIndent(report, "", "  ")
}

type junitReport struct {
	XMLName  xml.Name     `xml:"testsuites"`
	Name     string       `xml:"name,attr"`
	Suites   int          `xml:"suites,attr"`
	Tests    int          `xml:"tests,attr"`
	Failures int          `xml:"failures,attr"`
	Children []junitSuite `xml:"testsuite"`
}

type junitSuite struct {
	Name     string      `xml:"name,attr"`
	Tests    int         `xml:"tests,attr"`
	Failures int         `xml:"failures,attr"`
	Cases    []junitCase `xml:"testcase"`
}

type junitCase struct {
	Name    string     `xml:"name,attr"`
	Failure *junitFail `xml:"failure,omitempty"`
}

type junitFail struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

func RenderJUnit(report RunReport) ([]byte, error) {
	report.Recalculate()
	output := junitReport{
		Name:     "logstash-filter-verifier",
		Suites:   len(report.Suites),
		Tests:    report.Summary.TestCases,
		Failures: report.Summary.Failed,
		Children: make([]junitSuite, 0, len(report.Suites)),
	}

	for _, suite := range report.Suites {
		jSuite := junitSuite{
			Name:  suite.Name,
			Tests: len(suite.TestCases),
			Cases: make([]junitCase, 0, len(suite.TestCases)),
		}
		for _, testCase := range suite.TestCases {
			jCase := junitCase{Name: testCase.Name}
			if testCase.Status == StatusFailed {
				jSuite.Failures++
				jCase.Failure = &junitFail{
					Message: "Logstash event comparison failed",
					Body:    formatFailure(testCase),
				}
			}
			jSuite.Cases = append(jSuite.Cases, jCase)
		}
		output.Children = append(output.Children, jSuite)
	}

	body, err := xml.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), body...), nil
}

func formatFailure(testCase TestCaseReport) string {
	if testCase.Explain != "" {
		return strings.TrimSpace(testCase.Explain)
	}
	lines := make([]string, 0, len(testCase.Diffs))
	for _, diff := range testCase.Diffs {
		lines = append(lines, fmt.Sprintf("%s: %s", diff.Path, diff.Kind))
	}
	return strings.Join(lines, "\n")
}
