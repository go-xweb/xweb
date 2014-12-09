package xweb

import "github.com/go-xweb/log"

type LogInterface interface {
	SetLogger(*log.Logger)
}

type LogInterceptor struct {
	logger *log.Logger
}

func NewLogInterceptor(logger *log.Logger) *LogInterceptor {
	return &LogInterceptor{
		logger: logger,
	}
}

func (itor *LogInterceptor) Intercept(ai *Invocation) {
	action := ai.ActionContext().Action()
	if action != nil {
		if l, ok := action.(LogInterface); ok {
			l.SetLogger(itor.logger)
		}
	}

	ai.Invoke()

	if ai.Resp().Written() {
		statusCode := ai.Resp().StatusCode
		requestPath := ai.Req().URL.Path

		if statusCode >= 200 && statusCode < 400 {
			itor.logger.Info(ai.Req().Method, statusCode, requestPath)
		} else {
			itor.logger.Error(ai.Req().Method, statusCode, requestPath)
		}
	}
}
