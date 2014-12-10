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

func (itor *SessionInterceptor) Intercept(ctx *Context) {
	if action := ctx.Action(); ctx != nil {
		if s, ok := action.(SessionInterface); ok {
			session := itor.sessionMgr.Session(ctx.Req(), ctx.Resp())
			s.SetSessions(session)
		}
	}

	ctx.Invoke()
}
