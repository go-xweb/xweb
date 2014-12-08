package xweb

import "github.com/go-xweb/httpsession"

type SessionInterface interface {
	SetSessions(*httpsession.Session)
}

type SessionInterceptor struct {
	sessionMgr *httpsession.Manager
}

func NewSessionInterceptor(sessionMgr *httpsession.Manager) *SessionInterceptor {
	return &SessionInterceptor{sessionMgr: sessionMgr}
}

func (itor *SessionInterceptor) Intercept(ia *Invocation) {
	action := ia.ActionContext().Action()
	if s, ok := action.(SessionInterface); ok {
		session := itor.sessionMgr.Session(ia.Req(), ia.Resp())
		s.SetSessions(session)
	}

	ia.Invoke()
}
