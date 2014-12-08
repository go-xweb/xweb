package xweb

import (
	"net/http"
	"net/url"
	"regexp"
)

type Filter interface {
	Do(http.ResponseWriter, *http.Request) bool
}

func NewFilterInterceptor(filter Filter) *FilterInterceptor {
	return &FilterInterceptor{filter: filter}
}

type FilterInterceptor struct {
	filter Filter
}

func (itor *FilterInterceptor) Intercept(ia *Invocation) {
	if itor.filter != nil {
		if !itor.filter.Do(ia.Resp(), ia.Req()) {
			return
		}
	}

	ia.Invoke()
}

func (app *App) AddFilter(filter Filter) {
	app.interceptors = append(app.interceptors, NewFilterInterceptor(filter))
	//app.filters = append(app.filters, filter)
}

type LoginFilter struct {
	App           *App
	SessionName   string
	AnonymousUrls []*regexp.Regexp
	AskLoginUrls  []*regexp.Regexp
	Redirect      string
	OriUrlName    string
}

func NewLoginFilter(app *App, name string, redirect string) *LoginFilter {
	filter := &LoginFilter{App: app, SessionName: name,
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

	session := s.App.SessionManager.Session(req, w)
	id := session.Get(s.SessionName)
	has := (id != nil && id != "")

	var redirect = s.Redirect
	if s.OriUrlName != "" {
		redirect = redirect + "?" + s.OriUrlName + "=" + url.QueryEscape(req.URL.String())
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
			s.App.Redirect(w, requestPath, redirect)
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
		s.App.Redirect(w, requestPath, redirect)
	}
	return has
}
