package admin

import _ "embed"

var (
	//go:embed task-board.html
	taskBoardHTML []byte
)

// TaskBoardHTML returns the embedded admin task board page.
func TaskBoardHTML() []byte {
	return append([]byte(nil), taskBoardHTML...)
}
