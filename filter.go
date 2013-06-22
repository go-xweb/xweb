package xweb

import (
	"fmt"
	"net/http"
	"regexp"
)

type Filter interface {
	Do(http.ResponseWriter, *http.Request) bool
}

type LoginFilter struct {
	App           *App
	SessionName   string
	AnonymousUrls []*regexp.Regexp
	AskLoginUrls  []*regexp.Regexp
	Redirect      string
}

func NewLoginFilter(app *App, name string, redirect string) *LoginFilter {
	return &LoginFilter{App: app, SessionName: name,
		AnonymousUrls: make([]*regexp.Regexp, 0),
		AskLoginUrls:  make([]*regexp.Regexp, 0),
		Redirect:      redirect,
	}
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

func redirect(w http.ResponseWriter, url string) {
	w.Header().Set("Location", url)
	w.WriteHeader(302)
	w.Write([]byte("Redirecting to: " + url))
}

func (s *LoginFilter) Do(w http.ResponseWriter, req *http.Request) bool {
	requestPath := req.URL.Path
	fmt.Printf("LoginFilter: %v\n", requestPath)
	sess := s.App.SessionManager.SessionStart(w, req)
	defer sess.SessionRelease()

	id := sess.Get(s.SessionName)
	has := (id != nil)

	for _, cr := range s.AskLoginUrls {
		if !cr.MatchString(requestPath) {
			continue
		}
		match := cr.FindStringSubmatch(requestPath)
		if len(match[0]) != len(requestPath) {
			continue
		}
		if !has {
			redirect(w, s.Redirect)
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
		redirect(w, s.Redirect)
	}
	return has
}
