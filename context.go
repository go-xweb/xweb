package xweb

import (
	"net/http"
	"reflect"
)

type Context struct {
	router       *Router
	interceptors []Interceptor

	idx          int
	req          *http.Request
	resp         *ResponseWriter
	route        *Route
	args         []reflect.Value
	routeMatched bool

	action interface{}
	Result interface{}
}

func NewContext(
	router *Router,
	interceptors []Interceptor,
	req *http.Request,
	resp *ResponseWriter) *Context {
	return &Context{
		interceptors: interceptors,
		idx:          -1,
		req:          req,
		resp:         resp,
	}
}

func (ctx *Context) intercept() {
	ctx.interceptors[ctx.idx].Intercept(ctx)
}

func (ctx *Context) hasNext() bool {
	return (ctx.idx+1) >= 0 && (ctx.idx+1) < len(ctx.interceptors)
}

func (ctx *Context) next() {
	if ctx.idx >= len(ctx.interceptors)-1 {
		ctx.idx = -2
		return
	}
	ctx.idx += 1
}

func (ctx *Context) newAction() {
	if !ctx.routeMatched {
		reqPath := removeStick(ctx.Req().URL.Path)
		allowMethod := Ternary(ctx.Req().Method == "HEAD", "GET", ctx.Req().Method).(string)

		route, args := ctx.router.Match(reqPath, allowMethod)
		if route != nil {
			ctx.route = route
			ctx.action = route.newAction().Interface()
			ctx.args = args
		}
		ctx.routeMatched = true
	}
}

func (ctx *Context) Route() *Route {
	ctx.newAction()
	return ctx.route
}

func (ctx *Context) Action() interface{} {
	ctx.newAction()
	return ctx.action
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
	if ctx.hasNext() {
		ctx.next()
		ctx.intercept()
	} else {
		ctx.Result = ctx.Do()
	}
}

func (ctx *Context) Do() interface{} {
	ctx.newAction()
	if ctx.action == nil {
		return nil
	}

	var vc = reflect.ValueOf(ctx.action)
	function := vc.MethodByName(ctx.route.HandlerMethod)
	ret := function.Call(ctx.args)

	if len(ret) > 0 {
		return ret[0].Interface()
	}
	return nil
}
