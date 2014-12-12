package xweb

import (
	"net/http"
)

// handle return values
type ReturnInterceptor struct {
}

func (itor *ReturnInterceptor) Intercept(ctx *Context) {
	ctx.Invoke()

	// if has been write, then return
	if ctx.resp.Written() {
		return
	}

	if IsNil(ctx.Result) {
		if ctx.Action() == nil {
			// if there is no action match
			ctx.Result = NotFound()
		} else {
			// there is an action but return nil, then we return blank page
			ctx.Result = ""
		}
	}

	if err, ok := ctx.Result.(AbortError); ok {
		ctx.resp.WriteHeader(err.Code())
		ctx.resp.Write([]byte(err.Error()))
	} else if err, ok := ctx.Result.(error); ok {
		ctx.resp.WriteHeader(http.StatusInternalServerError)
		ctx.resp.Write([]byte(err.Error()))
	} else if bs, ok := ctx.Result.([]byte); ok {
		ctx.resp.WriteHeader(http.StatusOK)
		ctx.resp.Write(bs)
	} else if s, ok := ctx.Result.(string); ok {
		ctx.resp.WriteHeader(http.StatusOK)
		ctx.resp.Write([]byte(s))
	}
}
