package observer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/imkira/go-observer"

	"github.com/magnusbaeck/logstash-filter-verifier/v2/internal/reporting"
)

type ReportObserver struct {
	done       chan interface{}
	prop       observer.Property
	format     string
	outputPath string
	report     reporting.RunReport
}

func NewReportObserver(prop observer.Property, format, outputPath string) *ReportObserver {
	return &ReportObserver{
		done:       make(chan interface{}),
		prop:       prop,
		format:     format,
		outputPath: outputPath,
	}
}

func (ro *ReportObserver) Start() error {
	go func() {
		stream := ro.prop.Observe()
		for {
			switch event := stream.Value().(type) {
			case TestExecutionStart:
				ro.report = reporting.RunReport{
					Status:    reporting.StatusPassed,
					StartedAt: time.Now().UTC(),
					Suites:    make([]reporting.SuiteReport, 0),
				}
				_ = event
			case ComparisonResult:
				status := reporting.StatusPassed
				if !event.Status {
					status = reporting.StatusFailed
				}
				ro.addResult(event, status)
			case TestExecutionEnd:
				ro.report.Recalculate()
				if err := ro.write(); err != nil {
					fmt.Fprintf(os.Stderr, "failed to write report: %v\n", err)
				}
				ro.done <- true
				return
			}
			<-stream.Changes()
			stream.Next()
		}
	}()
	return nil
}

func (ro *ReportObserver) addResult(result ComparisonResult, status reporting.Status) {
	suiteIndex := -1
	for i := range ro.report.Suites {
		if ro.report.Suites[i].Name == result.Path {
			suiteIndex = i
			break
		}
	}
	if suiteIndex == -1 {
		ro.report.Suites = append(ro.report.Suites, reporting.SuiteReport{
			Name:      result.Path,
			Status:    reporting.StatusPassed,
			TestCases: make([]reporting.TestCaseReport, 0),
		})
		suiteIndex = len(ro.report.Suites) - 1
	}

	ro.report.Suites[suiteIndex].TestCases = append(
		ro.report.Suites[suiteIndex].TestCases,
		reporting.TestCaseReport{
			Name:       result.Name,
			Status:     status,
			EventIndex: result.EventIndex,
			Explain:    result.Explain,
			Diffs:      result.Diffs,
		},
	)
}

func (ro *ReportObserver) write() error {
	sort.Slice(ro.report.Suites, func(i, j int) bool {
		return ro.report.Suites[i].Name < ro.report.Suites[j].Name
	})
	for i := range ro.report.Suites {
		sort.Slice(ro.report.Suites[i].TestCases, func(a, b int) bool {
			if ro.report.Suites[i].TestCases[a].EventIndex == ro.report.Suites[i].TestCases[b].EventIndex {
				return ro.report.Suites[i].TestCases[a].Name < ro.report.Suites[i].TestCases[b].Name
			}
			return ro.report.Suites[i].TestCases[a].EventIndex < ro.report.Suites[i].TestCases[b].EventIndex
		})
	}

	var body []byte
	var err error
	switch ro.format {
	case "json":
		body, err = reporting.RenderJSON(ro.report)
	case "junit":
		body, err = reporting.RenderJUnit(ro.report)
	default:
		return fmt.Errorf("unsupported report format %q", ro.format)
	}
	if err != nil {
		return err
	}

	if ro.outputPath == "" {
		_, err = os.Stdout.Write(append(body, '\n'))
		return err
	}
	if err = os.MkdirAll(filepath.Dir(ro.outputPath), 0700); err != nil {
		return err
	}
	return os.WriteFile(ro.outputPath, body, 0600)
}

func (ro *ReportObserver) Finalize() error {
	<-ro.done
	return nil
}
