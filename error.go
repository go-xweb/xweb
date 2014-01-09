package xweb

import (
	"fmt"
	"net/http"
)

type AbortError struct {
	Code    int
	Content string
}

func (a *AbortError) Error() string {
	return fmt.Sprintf("%v %v", a.Code, a.Content)
}

func Abort(code int, content ...string) error {
	if len(content) >= 1 {
		return &AbortError{code, content[0]}
	}
	return &AbortError{code, statusText[code]}
}

func NotFound(content ...string) error {
	return Abort(http.StatusNotFound, content...)
}

func NotSupported(content ...string) error {
	return Abort(http.StatusMethodNotAllowed, content...)
}

func InternalServerError(content ...string) error {
	return Abort(http.StatusInternalServerError, content...)
}

func Forbidden(content ...string) error {
	return Abort(http.StatusForbidden, content...)
}

func Unauthorized(content ...string) error {
	return Abort(http.StatusUnauthorized, content...)
}
