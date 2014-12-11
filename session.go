package xweb

import (
	"time"

	"github.com/go-xweb/httpsession"
)

type SessionInterface interface {
	SetSessions(*httpsession.Session)
}

type Sessions struct {
	*httpsession.Manager
}

func NewSessions(sessionMgr *httpsession.Manager, sessionTimeout time.Duration) *Sessions {
	if sessionMgr == nil {
		sessionMgr = httpsession.Default()
	}
	if sessionTimeout > time.Second {
		sessionMgr.SetMaxAge(sessionTimeout)
	}
	sessionMgr.Run()

	return &Sessions{Manager: sessionMgr}
}

func (itor *Sessions) Intercept(ctx *Context) {
	if action := ctx.Action(); ctx != nil {
		if s, ok := action.(SessionInterface); ok {
			session := itor.Session(ctx.Req(), ctx.Resp())
			s.SetSessions(session)
		}
	}

	ctx.Invoke()
}
