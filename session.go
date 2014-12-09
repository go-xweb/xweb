package xweb

import (
	"time"

	"github.com/go-xweb/httpsession"
)

type SessionInterface interface {
	SetSessions(*httpsession.Session)
}

type SessionInterceptor struct {
	sessionMgr *httpsession.Manager
}

func NewSessionInterceptor(app *App) *SessionInterceptor {
	if app.Server.SessionManager != nil {
		app.SessionManager = app.Server.SessionManager
	} else {
		app.SessionManager = httpsession.Default()
		if app.AppConfig.SessionTimeout > time.Second {
			app.SessionManager.SetMaxAge(app.AppConfig.SessionTimeout)
		}
		app.SessionManager.Run()
	}

	return &SessionInterceptor{sessionMgr: app.SessionManager}
}

func (itor *SessionInterceptor) Intercept(ia *Invocation) {
	action := ia.ActionContext().Action()
	if s, ok := action.(SessionInterface); ok {
		session := itor.sessionMgr.Session(ia.Req(), ia.Resp())
		s.SetSessions(session)
	}

	ia.Invoke()
}
