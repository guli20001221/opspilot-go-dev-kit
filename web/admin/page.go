package admin

import _ "embed"

var (
	//go:embed task-board.html
	taskBoardHTML []byte
	//go:embed reports.html
	reportsHTML []byte
)

// TaskBoardHTML returns the embedded admin task board page.
func TaskBoardHTML() []byte {
	return append([]byte(nil), taskBoardHTML...)
}

// ReportsHTML returns the embedded admin reports page.
func ReportsHTML() []byte {
	return append([]byte(nil), reportsHTML...)
}
