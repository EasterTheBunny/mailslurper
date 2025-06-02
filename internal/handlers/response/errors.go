package response

import "errors"

var (
	ErrIncorrectRoute         = errors.New("incorrect routing")
	ErrUnsupportedContentType = errors.New("unsupported content type")
	ErrMethodNotAllowed       = errors.New("method not allowed")
	ErrNotFound               = errors.New("not found")
	ErrInvalidInput           = errors.New("invalid input")
	ErrMissingFormTag         = errors.New("struct missing tag 'form'")
	ErrMissingValueForTag     = errors.New("missing value for tag 'form'")
	ErrNotAPointer            = errors.New("not a pointer")
	ErrUnexpectedDataType     = errors.New("unexpected data type")
)
