package xweb

type Logger interface {
	Debugf(format string, v ...interface{})
	Debug(v ...interface{})
	Infof(format string, v ...interface{})
	Info(v ...interface{})
	Warnf(format string, v ...interface{})
	Warn(v ...interface{})
	Errorf(format string, v ...interface{})
	Error(v ...interface{})
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
	Panic(v ...interface{})
	Panicf(format string, v ...interface{})
}

type LogInterface interface {
	SetLogger(Logger)
}

type LogInterceptor struct {
	logger Logger
}

func NewLogInterceptor(logger Logger) *LogInterceptor {
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
