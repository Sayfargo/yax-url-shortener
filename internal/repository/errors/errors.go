package repoerrors

import (
	"errors"
	"fmt"
)

var (
	ErrNotExists         = errors.New("row not found in db")
	ErrConflictShortCode = errors.New("short code already exists")
	ErrUnexpectedType    = errors.New("unexpected data type in cache")
)

type BatchConflictError struct {
	Index int
	Err   error
}

func (e *BatchConflictError) Error() string {
	return fmt.Sprintf("conflict at index %d: %v", e.Index, e.Err)
}

type OriginalUrlConflictError struct {
	ShortCode string
}

func (o *OriginalUrlConflictError) Error() string {
	return fmt.Sprintf("conflict at url: %s", o.ShortCode)
}
