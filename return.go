package xweb

// handle return values
type ReturnInterceptor struct {
}

func (itor *ReturnInterceptor) Intercept(ctx *Context) {
	ctx.Invoke()

	// if no any return status code
	if !ctx.Resp().Written() {
		if ctx.Result == nil {
			if ctx.Action() == nil {
				ctx.Result = NotFound()
			} else {
				ctx.Result = ""
			}
		}
		ctx.HandleResult(ctx.Result)
	}
}
