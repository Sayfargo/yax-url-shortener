package repository_errors

import "errors"

var (
	ErrNotExists      = errors.New("row not found in db")
	ErrAlreadyExists  = errors.New("row already exists")
	ErrUnexpectedType = errors.New("unexpected data type in cache")
)
