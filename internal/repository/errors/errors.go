package repository_errors

import (
	"errors"
	"fmt"
)

var (
	ErrNotExists      = errors.New("row not found in db")
	ErrAlreadyExists  = errors.New("row already exists")
	ErrUnexpectedType = errors.New("unexpected data type in cache")
)

type BatchConflictError struct {
	Index int
	Err   error
}

func (e *BatchConflictError) Error() string {
	return fmt.Sprintf("conflict at index %d: %v", e.Index, e.Err)
}
