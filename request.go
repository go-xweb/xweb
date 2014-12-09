package xweb

import "net/http"

type RequestInterface interface {
	SetRequest(*http.Request)
}

type RequestInterceptor struct {
}

func (ii *RequestInterceptor) Intercept(ia *Invocation) {
	action := ia.ActionContext().Action()
	if s, ok := action.(RequestInterface); ok {
		s.SetRequest(ia.Req())
	}

	ia.Invoke()
}
