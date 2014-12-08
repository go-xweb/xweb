package xweb

type LogInterceptor struct {
}

func (itor *LogInterceptor) Intercept(ai *Invocation) {
	ai.Invoke()

	if ai.Resp().Written() {
		statusCode := ai.Resp().StatusCode
		requestPath := ai.Req().URL.Path

		if statusCode >= 200 && statusCode < 400 {
			ai.app.Info(ai.Req().Method, statusCode, requestPath)
		} else {
			ai.app.Error(ai.Req().Method, statusCode, requestPath)
		}
	}
}
