package xweb

import "github.com/go-xweb/log"

type LogInterface interface {
	SetLogger(*log.Logger)
}

type LogInterceptor struct {
}

func (itor *LogInterceptor) Intercept(ai *Invocation) {
	action := ai.ActionContext().Action()
	if action != nil {
		if l, ok := action.(LogInterface); ok {
			l.SetLogger(ai.app.Logger)
		}
	}

	ai.Invoke()

	if ai.Resp().Written() {
		statusCode := ai.Resp().StatusCode
		requestPath := ai.Req().URL.Path

		if statusCode >= 200 && statusCode < 400 {
			ai.app.Logger.Info(ai.Req().Method, statusCode, requestPath)
		} else {
			ai.app.Logger.Error(ai.Req().Method, statusCode, requestPath)
		}
	}
}
