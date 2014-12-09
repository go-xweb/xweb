package xweb

import (
	"fmt"
	"net/http"
	"runtime"
)

type PanicInterceptor struct {
	debug bool
}

func NewPanicInterceptor(isDebug bool) *PanicInterceptor {
	return &PanicInterceptor{debug: isDebug}
}

func (itor *PanicInterceptor) Intercept(ia *Invocation) {
	defer func() {
		if e := recover(); e != nil {
			if !ia.app.Server.Config.RecoverPanic {
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

				ia.app.Logger.Error(content)

				ia.Resp().WriteHeader(http.StatusInternalServerError)
				if !itor.debug {
					content = statusText[http.StatusInternalServerError]
				}
				ia.Resp().Write([]byte(content))
			}
		}
	}()

	ia.Invoke()
}
