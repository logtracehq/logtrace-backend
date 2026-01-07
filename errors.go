package logbase

import "strings"

var ErrDuplicateRecord = "duplicate key value violates unique constraint"

type LogbaseError string

func (m LogbaseError) Error() string { return string(m) }

func IsDuplicateUniqueError(e error) bool {
	if e == nil {
		return false
	}

	return strings.Contains(e.Error(), "duplicate key value violates unique constraint")
}
