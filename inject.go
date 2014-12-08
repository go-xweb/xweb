package xweb

import "net/http"

type AppInterface interface {
	SetApp(*App)
}

type RequestInterface interface {
	SetRequest(*http.Request)
}

type ResponseInterface interface {
	SetResponse(*ResponseWriter)
}

type InjectInterceptor struct {
}

func (ii *InjectInterceptor) Intercept(ia *Invocation) {
	action := ia.ActionContext().Action()
	if s, ok := action.(RequestInterface); ok {
		s.SetRequest(ia.Req())
	}

	if s, ok := action.(ResponseInterface); ok {
		s.SetResponse(ia.Resp())
	}

	if s, ok := action.(AppInterface); ok {
		s.SetApp(ia.app)
	}

	ia.Invoke()
}
