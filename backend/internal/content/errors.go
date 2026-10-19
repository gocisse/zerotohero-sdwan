package content

import (
	"fmt"
	"strings"
)

// Error is one problem found in the content, tied to a file (relative to the content dir).
type Error struct {
	File    string `json:"file"`
	Message string `json:"message"`
}

func (e Error) Error() string { return e.File + ": " + e.Message }

// ErrorList collects every problem found so they can be reported together.
type ErrorList []Error

func (l ErrorList) Error() string {
	lines := make([]string, len(l))
	for i, e := range l {
		lines[i] = e.Error()
	}
	return fmt.Sprintf("%d content error(s):\n  %s", len(l), strings.Join(lines, "\n  "))
}

func (l *ErrorList) add(file, format string, args ...any) {
	*l = append(*l, Error{File: file, Message: fmt.Sprintf(format, args...)})
}
