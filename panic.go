package xweb

import (
	"fmt"
	"net/http"
	"runtime"
)

type PanicInterceptor struct {
	recoverPanic bool
	debug        bool
	logger       Logger
}

func (inter *PanicInterceptor) SetLogger(logger Logger) {
	inter.logger = logger
}

func NewPanicInterceptor(recoverPanic, isDebug bool) *PanicInterceptor {
	return &PanicInterceptor{debug: isDebug}
}

func (itor *PanicInterceptor) Intercept(ctx *Context) {
	defer func() {
		if e := recover(); e != nil {
			if !itor.recoverPanic {
				// go back to panic
				panic(e)
			} else {
				var content string
				content = fmt.Sprintf("Handler crashed with error: %v", e)
				for i := 1; ; i += 1 {
					_, file, line, ok := runtime.Caller(i)
					if !ok {
						break
					} else {
						content += "\n"
					}
					content += fmt.Sprintf("%v %v", file, line)
				}

				itor.logger.Error(content)

				ctx.Resp().WriteHeader(http.StatusInternalServerError)
				if !itor.debug {
					content = statusText[http.StatusInternalServerError]
				}
				ctx.Resp().Write([]byte(content))
			}
		}
	}()

	ctx.Invoke()
}
