package xweb

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/go-xweb/httpsession"
)

type Interceptor interface {
	Intercept(*Invocation)
}

type Invocation struct {
	interceptors   []Interceptor
	idx            int
	action         *ActionContext
	req            *http.Request
	resp           *ResponseWriter
	SessionManager *httpsession.Manager

	Result interface{}
}

func NewInvocation(interceptors []Interceptor, req *http.Request,
	resp *ResponseWriter, ac *ActionContext) *Invocation {
	return &Invocation{
		interceptors: interceptors,
		idx:          -1,
		action:       ac,
		req:          req,
		resp:         resp,
	}
}

func (invocation *Invocation) Req() *http.Request {
	return invocation.req
}

func (invocation *Invocation) Resp() *ResponseWriter {
	return invocation.resp
}

func (invocation *Invocation) ServeFile(path string) error {
	return invocation.resp.ServeFile(invocation.req, path)
}

func (invocation *Invocation) ActionContext() *ActionContext {
	return invocation.action
}

func (invocation *Invocation) Interceptor() Interceptor {
	return invocation.interceptors[invocation.idx]
}

func (invocation *Invocation) HasNext() bool {
	return (invocation.idx+1) >= 0 && (invocation.idx+1) < len(invocation.interceptors)
}

func (invocation *Invocation) Next() {
	if invocation.idx >= len(invocation.interceptors)-1 {
		invocation.idx = -2
		return
	}
	invocation.idx += 1
}

func (invocation *Invocation) HandleResult(result interface{}) bool {
	if IsNil(result) {
		return false
	}

	if err, ok := result.(AbortError); ok {
		if !invocation.resp.Written() {
			invocation.resp.WriteHeader(err.Code())
			invocation.resp.Write([]byte(err.Error()))
		}
	} else if err, ok := result.(error); ok {
		if !invocation.resp.Written() {
			invocation.resp.WriteHeader(http.StatusInternalServerError)
			invocation.resp.Write([]byte(err.Error()))
		} else {
			// TODO: log the error
		}
		return true
	} else if bs, ok := result.([]byte); ok {
		invocation.resp.Write(bs)
		return true
	} else if s, ok := result.(string); ok {
		invocation.resp.Write([]byte(s))
		return true
	}
	return false
}

func (invocation *Invocation) Invoke() {
	if invocation.HasNext() {
		invocation.Next()
		invocation.Interceptor().Intercept(invocation)
	} else {
		ac := invocation.ActionContext()
		invocation.Result = ac.Execute()
	}
}

type StaticInterceptor struct {
	RootPath   string
	IndexFiles []string
}

func (itor *StaticInterceptor) serveFile(ai *Invocation, path string) bool {
	fPath := filepath.Join(itor.RootPath, path)
	finfo, err := os.Stat(fPath)
	if err != nil {
		if !os.IsNotExist(err) {
			ai.HandleResult(err)
			return true
		}
	} else if !finfo.IsDir() {
		err := ai.ServeFile(fPath)
		if err != nil {
			ai.HandleResult(err)
		}
		return true
	}
	return false
}

func (itor *StaticInterceptor) Intercept(ai *Invocation) {
	if ai.Req().Method == "GET" || ai.Req().Method == "HEAD" {
		if itor.serveFile(ai, ai.Req().URL.Path) {
			return
		}
	}

	ai.Invoke()

	// try serving index.html or index.htm
	if !ai.Resp().Written() && (ai.Req().Method == "GET" || ai.Req().Method == "HEAD") {
		if len(itor.IndexFiles) > 0 {
			for _, index := range itor.IndexFiles {
				if itor.serveFile(ai, path.Join(ai.Req().URL.Path, index)) {
					return
				}
			}
		}
	}
}

type BeforeInterface interface {
	Before(structName, actionName string) bool
}

type BeforeInterceptor struct {
}

func (itor *BeforeInterceptor) Intercept(ai *Invocation) {
	action := ai.ActionContext().Action()
	if action == nil {
		return
	}

	if before, ok := action.(BeforeInterface); ok {
		route := ai.ActionContext().getRoute()
		if !before.Before(route.HandlerElement.Name(),
			route.HandlerMethod) {
			return
		}
	}
	ai.Invoke()
}

type AfterInterface interface {
	After(structName, actionName string, result interface{}) bool
}

type AfterInterceptor struct {
}

func (itor *AfterInterceptor) Intercept(ai *Invocation) {
	ai.Invoke()

	action := ai.ActionContext().Action()
	if action == nil {
		return
	}

	if after, ok := action.(AfterInterface); ok {
		route := ai.ActionContext().getRoute()
		if !after.After(route.HandlerElement.Name(),
			route.HandlerMethod, ai.Result) {
			fmt.Println("since we return false, but I cannot stop the other interceptors")
		}
	}
}

type InitInterface interface {
	Init()
}

type InitInterceptor struct {
}

func (itor *InitInterceptor) Intercept(ai *Invocation) {
	if init, ok := ai.ActionContext().Action().(InitInterface); ok {
		init.Init()
	}
	ai.Invoke()
}
