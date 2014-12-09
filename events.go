package xweb

import "fmt"

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
