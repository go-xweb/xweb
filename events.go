package xweb

import (
	"fmt"

	"github.com/lunny/tango"
)

type BeforeInterface interface {
	Before(structName, actionName string) bool
}

type AfterInterface interface {
	After(structName, actionName string, result interface{}) bool
}

type InitInterface interface {
	Init()
}

func NewEventsHandle() tango.Handler {
	return tango.HandlerFunc(EventsHandle)
}

func EventsHandle(ctx *tango.Context) {
	action := ctx.Action()
	if action != nil {
		if init, ok := action.(InitInterface); ok {
			init.Init()
		}

		if before, ok := action.(BeforeInterface); ok {
			route := ctx.Route()
			if !before.Before(route.StructType().Name(),
				route.Method().Type().Name()) {
				return
			}
		}
	}

	ctx.Next()

	if action == nil {
		return
	}

	if after, ok := action.(AfterInterface); ok {
		route := ctx.Route()
		if !after.After(
			route.StructType().Name(),
			route.Method().Type().Name(),
			ctx.Result) {
			fmt.Println("we current cannot disallow invoke to next interceptors")
		}
	}
}
