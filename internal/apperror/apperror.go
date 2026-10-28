package apperror

import "errors"

type Kind int

const (
	KindInternal Kind = iota
	KindInvalid
	KindNotFound
	KindConflict
	KindValidation
)

type FieldError struct {
	Field  string
	Reason string
}

type Error struct {
	Kind   Kind
	Detail string
	Fields []FieldError
	Err    error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Detail + ": " + e.Err.Error()
	}
	return e.Detail
}

func (e *Error) Unwrap() error {
	return e.Err
}

func Invalid(detail string) *Error {
	return &Error{Kind: KindInvalid, Detail: detail}
}

func NotFound(detail string) *Error {
	return &Error{Kind: KindNotFound, Detail: detail}
}

func Conflict(detail string, err error) *Error {
	return &Error{Kind: KindConflict, Detail: detail, Err: err}
}

func Validation(fields []FieldError) *Error {
	return &Error{Kind: KindValidation, Detail: "campo(s) invalidos", Fields: fields}
}

func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return KindInternal
}
