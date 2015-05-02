package xweb

import (
	"net/http"
	"net/url"
	"regexp"

	"github.com/go-xweb/httpsession"
	"github.com/lunny/tango"
)

type Filter interface {
	Do(http.ResponseWriter, *http.Request) bool
}

func FilterHandler(filter Filter) tango.HandlerFunc {
	return func(ctx *tango.Context) {
		if filter != nil {
			if !filter.Do(ctx, ctx.Req()) {
				return
			}
		}

		ctx.Next()
	}
}

// for compitable
func (app *App) AddFilter(filter Filter) {
	app.Use(FilterHandler(filter))
}

type LoginFilter struct {
	SessionName   string
	sessionMgr    *httpsession.Manager
	AnonymousUrls []*regexp.Regexp
	AskLoginUrls  []*regexp.Regexp
	Redirect      string
	OriUrlName    string
}

func (s *LoginFilter) SetSessionMgr(sessionMgr *httpsession.Manager) {
	s.sessionMgr = sessionMgr
}

func NewLoginFilter(app *App, name string, redirect string) *LoginFilter {
	filter := &LoginFilter{
		SessionName:   name,
		AnonymousUrls: make([]*regexp.Regexp, 0),
		AskLoginUrls:  make([]*regexp.Regexp, 0),
		Redirect:      redirect,
	}
	filter.AddAnonymousUrls("/favicon.ico", redirect)
	return filter
}

func (s *LoginFilter) AddAnonymousUrls(urls ...string) {
	for _, r := range urls {
		cr, err := regexp.Compile(r)
		if err == nil {
			s.AnonymousUrls = append(s.AnonymousUrls, cr)
		}
	}
}

func (s *LoginFilter) AddAskLoginUrls(urls ...string) {
	for _, r := range urls {
		cr, err := regexp.Compile(r)
		if err == nil {
			s.AskLoginUrls = append(s.AskLoginUrls, cr)
		}
	}
}

func (s *LoginFilter) Do(w http.ResponseWriter, req *http.Request) bool {
	requestPath := removeStick(req.URL.Path)

	session := s.sessionMgr.Session(req, w)
	id := session.Get(s.SessionName)
	has := (id != nil && id != "")

	var rd = s.Redirect
	if s.OriUrlName != "" {
		rd = rd + "?" + s.OriUrlName + "=" + url.QueryEscape(req.URL.String())
	}

	for _, cr := range s.AskLoginUrls {
		if !cr.MatchString(requestPath) {
			continue
		}
		match := cr.FindStringSubmatch(requestPath)
		if len(match[0]) != len(requestPath) {
			continue
		}
		if !has {
			redirect(w, rd)
		}
		return has
	}

	if len(s.AnonymousUrls) == 0 {
		return true
	}

	for _, cr := range s.AnonymousUrls {
		if !cr.MatchString(requestPath) {
			continue
		}
		match := cr.FindStringSubmatch(requestPath)
		if len(match[0]) != len(requestPath) {
			continue
		}
		return true
	}

	if !has {
		redirect(w, rd)
	}
	return has
}
