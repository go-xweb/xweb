package xweb

import "fmt"

type BeforeInterface interface {
	Before(structName, actionName string) bool
}

type BeforeInterceptor struct {
}

func (itor *BeforeInterceptor) Intercept(ctx *Context) {
	if action := ctx.Action(); action != nil {
		if before, ok := action.(BeforeInterface); ok {
			route := ctx.getRoute()
			if !before.Before(route.HandlerElement.Name(),
				route.HandlerMethod) {
				return
			}
		}
	}
	ctx.Invoke()
}

type AfterInterface interface {
	After(structName, actionName string, result interface{}) bool
}

type AfterInterceptor struct {
}

func (itor *AfterInterceptor) Intercept(ctx *Context) {
	ctx.Invoke()

	action := ctx.Action()
	if action == nil {
		return
	}

	if after, ok := action.(AfterInterface); ok {
		route := ctx.getRoute()
		if !after.After(
			route.HandlerElement.Name(),
			route.HandlerMethod,
			ctx.Result) {
			fmt.Println("we current cannot disallow invoke to next interceptors")
		}
	}
}

type InitInterface interface {
	Init()
}

type InitInterceptor struct {
}

func (itor *InitInterceptor) Intercept(ctx *Context) {
	if action := ctx.Action(); action != nil {
		if init, ok := action.(InitInterface); ok {
			init.Init()
		}
	}
	ctx.Invoke()
}
