package xweb

import (
	"net/http"
)

type AbortError struct {
	Code    int
	Content string
}

func (a *AbortError) Error() string {
	return a.Content
}

/*func Error(w ResponseWriter, err error) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	fmt.Fprintln(w, err)
}*/

func Abort(code int, content ...string) error {
	if len(content) >= 1 {
		return &AbortError{code, content[0]}
	}
	return &AbortError{code, statusText[code]}
}

func NotFound(content ...string) error {
	return Abort(http.StatusNotFound)
}

func NotSupported(content ...string) error {
	return Abort(http.StatusMethodNotAllowed)
}

func InterError(content ...string) error {
	return Abort(http.StatusInternalServerError)
}

func Forbidden(content ...string) error {
	return Abort(http.StatusForbidden)
}

func Unauthorized(content ...string) error {
	return Abort(http.StatusUnauthorized)
}
