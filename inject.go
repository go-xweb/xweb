package xweb

import (
	"net/http"

	"github.com/go-xweb/httpsession"
)

type SessionInterface interface {
	SetSessions(*httpsession.Session)
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
	if s, ok := action.(SessionInterface); ok {
		session := ia.SessionManager.Session(ia.Req(), ia.Resp())
		s.SetSessions(session)
	}

	if s, ok := action.(RequestInterface); ok {
		s.SetRequest(ia.Req())
	}

	if s, ok := action.(ResponseInterface); ok {
		s.SetResponse(ia.Resp())
	}

	ia.Invoke()
}
