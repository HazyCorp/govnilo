package hazycheck

import "errors"

type InternalErr struct {
	Internal error
}

func InternalError(err error) error {
	return &InternalErr{Internal: err}
}

func (e *InternalErr) Error() string {
	return e.Internal.Error()
}

var ErrNotFound = errors.New("entity not found")
