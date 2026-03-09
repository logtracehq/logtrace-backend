package logtrace

import "strings"

var ErrDuplicateRecord = "duplicate key value violates unique constraint"

type LogtraceError string

func (m LogtraceError) Error() string { return string(m) }

func IsDuplicateUniqueError(e error) bool {
	if e == nil {
		return false
	}

	return strings.Contains(e.Error(), "duplicate key value violates unique constraint")
}
