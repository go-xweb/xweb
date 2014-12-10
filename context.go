package xweb

import "net/http"

type Context struct {
	*Injector

	interceptors []Interceptor
	idx          int
	req          *http.Request
	resp         *ResponseWriter
	route        *Route
	routeMatched bool

	action    interface{}
	Execute   func() interface{}
	newAction func()

	Result interface{}
}

func NewContext(injector *Injector,
	interceptors []Interceptor,
	req *http.Request,
	resp *ResponseWriter) *Context {
	return &Context{
		Injector:     injector,
		interceptors: interceptors,
		idx:          -1,
		req:          req,
		resp:         resp,
	}
}

func (ctx *Context) Req() *http.Request {
	return ctx.req
}

func (ctx *Context) Resp() *ResponseWriter {
	return ctx.resp
}

func (ctx *Context) ServeFile(path string) error {
	return ctx.resp.ServeFile(ctx.req, path)
}

func (ctx *Context) Interceptor() Interceptor {
	return ctx.interceptors[ctx.idx]
}

func (ctx *Context) HasNext() bool {
	return (ctx.idx+1) >= 0 && (ctx.idx+1) < len(ctx.interceptors)
}

func (ctx *Context) next() {
	if ctx.idx >= len(ctx.interceptors)-1 {
		ctx.idx = -2
		return
	}
	ctx.idx += 1
}

func (ctx *Context) HandleResult(result interface{}) bool {
	if IsNil(result) {
		return false
	}

	if ctx.resp.Written() {
		return true
	}

	if err, ok := result.(AbortError); ok {
		ctx.resp.WriteHeader(err.Code())
		ctx.resp.Write([]byte(err.Error()))
		return true
	} else if err, ok := result.(error); ok {
		ctx.resp.WriteHeader(http.StatusInternalServerError)
		ctx.resp.Write([]byte(err.Error()))
		return true
	} else if bs, ok := result.([]byte); ok {
		ctx.resp.WriteHeader(http.StatusOK)
		ctx.resp.Write(bs)
		return true
	} else if s, ok := result.(string); ok {
		ctx.resp.WriteHeader(http.StatusOK)
		ctx.resp.Write([]byte(s))
		return true
	}
	return false
}

func (ctx *Context) Invoke() {
	if ctx.HasNext() {
		ctx.next()
		ctx.Interceptor().Intercept(ctx)
	} else {
		ctx.Result = ctx.Execute()
	}
}

func (ctx *Context) Action() interface{} {
	if ctx.action == nil && ctx.newAction != nil {
		ctx.newAction()
	}
	return ctx.action
}

func (ctx *Context) getRoute() *Route {
	if ctx.action == nil && ctx.newAction != nil {
		ctx.newAction()
	}
	return ctx.route
}
