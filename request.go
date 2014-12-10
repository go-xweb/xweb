package xweb

import "net/http"

type RequestInterface interface {
	SetRequest(*http.Request)
}

type RequestInterceptor struct {
}

func (ii *RequestInterceptor) Intercept(ctx *Context) {
	if action := ctx.Action(); action != nil {
		if s, ok := action.(RequestInterface); ok {
			s.SetRequest(ctx.Req())
		}
	}

	ctx.Invoke()
}
