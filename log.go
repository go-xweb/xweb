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

func (itor *LogInterceptor) Intercept(ctx *Context) {
	if action := ctx.Action(); action != nil {
		if l, ok := action.(LogInterface); ok {
			l.SetLogger(itor.logger)
		}
	}

	ctx.Invoke()

	if ctx.Resp().Written() {
		statusCode := ctx.Resp().StatusCode
		requestPath := ctx.Req().URL.Path

		if statusCode >= 200 && statusCode < 400 {
			itor.logger.Info(ctx.Req().Method, statusCode, requestPath)
		} else {
			itor.logger.Error(ctx.Req().Method, statusCode, requestPath)
		}
	}
}
