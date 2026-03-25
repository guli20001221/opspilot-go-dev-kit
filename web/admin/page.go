package admin

import _ "embed"

var (
	//go:embed task-board.html
	taskBoardHTML []byte
	//go:embed cases.html
	casesHTML []byte
	//go:embed evals.html
	evalsHTML []byte
	//go:embed eval-datasets.html
	evalDatasetsHTML []byte
	//go:embed reports.html
	reportsHTML []byte
	//go:embed report-compare.html
	reportCompareHTML []byte
	//go:embed trace-detail.html
	traceDetailHTML []byte
	//go:embed version-detail.html
	versionDetailHTML []byte
)

// TaskBoardHTML returns the embedded admin task board page.
func TaskBoardHTML() []byte {
	return append([]byte(nil), taskBoardHTML...)
}

// CasesHTML returns the embedded admin cases page.
func CasesHTML() []byte {
	return append([]byte(nil), casesHTML...)
}

// EvalsHTML returns the embedded admin evals page.
func EvalsHTML() []byte {
	return append([]byte(nil), evalsHTML...)
}

// EvalDatasetsHTML returns the embedded admin eval datasets page.
func EvalDatasetsHTML() []byte {
	return append([]byte(nil), evalDatasetsHTML...)
}

// ReportsHTML returns the embedded admin reports page.
func ReportsHTML() []byte {
	return append([]byte(nil), reportsHTML...)
}

// ReportCompareHTML returns the embedded admin report comparison page.
func ReportCompareHTML() []byte {
	return append([]byte(nil), reportCompareHTML...)
}

// TraceDetailHTML returns the embedded admin trace detail page.
func TraceDetailHTML() []byte {
	return append([]byte(nil), traceDetailHTML...)
}

// VersionDetailHTML returns the embedded admin version detail page.
func VersionDetailHTML() []byte {
	return append([]byte(nil), versionDetailHTML...)
}
