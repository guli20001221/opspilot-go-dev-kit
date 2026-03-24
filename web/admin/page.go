package admin

import _ "embed"

var (
	//go:embed task-board.html
	taskBoardHTML []byte
	//go:embed cases.html
	casesHTML []byte
	//go:embed reports.html
	reportsHTML []byte
	//go:embed report-compare.html
	reportCompareHTML []byte
	//go:embed trace-detail.html
	traceDetailHTML []byte
)

// TaskBoardHTML returns the embedded admin task board page.
func TaskBoardHTML() []byte {
	return append([]byte(nil), taskBoardHTML...)
}

// CasesHTML returns the embedded admin cases page.
func CasesHTML() []byte {
	return append([]byte(nil), casesHTML...)
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
