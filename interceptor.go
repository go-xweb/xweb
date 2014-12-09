package xweb

import "net/http"

type Interceptor interface {
	Intercept(*Invocation)
}

type Invocation struct {
	*Injector
	interceptors []Interceptor
	idx          int
	action       *ActionContext
	req          *http.Request
	resp         *ResponseWriter

	Result interface{}
}

func NewInvocation(injector *Injector, interceptors []Interceptor, req *http.Request,
	resp *ResponseWriter, ac *ActionContext) *Invocation {
	return &Invocation{
		Injector:     injector,
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

	if invocation.resp.Written() {
		return true
	}

	if err, ok := result.(AbortError); ok {
		invocation.resp.WriteHeader(err.Code())
		invocation.resp.Write([]byte(err.Error()))
		return true
	} else if err, ok := result.(error); ok {
		invocation.resp.WriteHeader(http.StatusInternalServerError)
		invocation.resp.Write([]byte(err.Error()))
		return true
	} else if bs, ok := result.([]byte); ok {
		invocation.resp.WriteHeader(http.StatusOK)
		invocation.resp.Write(bs)
		return true
	} else if s, ok := result.(string); ok {
		invocation.resp.WriteHeader(http.StatusOK)
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

// handle return values
type ReturnInterceptor struct {
}

func (itor *ReturnInterceptor) Intercept(ai *Invocation) {
	ai.Invoke()

	// if no any return status code
	if !ai.Resp().Written() {
		if ai.Result == nil {
			if ai.ActionContext().Action() == nil {
				ai.Result = NotFound()
			} else {
				ai.Result = ""
			}
		}
		ai.HandleResult(ai.Result)
	}
}
