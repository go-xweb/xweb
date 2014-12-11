package xweb

import "fmt"

type BeforeInterface interface {
	Before(structName, actionName string) bool
}

type AfterInterface interface {
	After(structName, actionName string, result interface{}) bool
}

type InitInterface interface {
	Init()
}

type Events struct {
}

func (itor *Events) Intercept(ctx *Context) {
	action := ctx.Action()
	if action != nil {
		if init, ok := action.(InitInterface); ok {
			init.Init()
		}

		if before, ok := action.(BeforeInterface); ok {
			route := ctx.Route()
			if !before.Before(route.HandlerElement.Name(),
				route.HandlerMethod) {
				return
			}
		}
	}

	ctx.Invoke()

	if action == nil {
		return
	}

	if after, ok := action.(AfterInterface); ok {
		route := ctx.Route()
		if !after.After(
			route.HandlerElement.Name(),
			route.HandlerMethod,
			ctx.Result) {
			fmt.Println("we current cannot disallow invoke to next interceptors")
		}
	}
}