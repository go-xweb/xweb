package xweb

import (
	"fmt"
	"reflect"

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

func Events() tango.HandlerFunc {
	return func(ctx *tango.Context) {
		action := ctx.Action()
		if action != nil {
			if init, ok := action.(InitInterface); ok {
				init.Init()
			}

			if before, ok := action.(BeforeInterface); ok {
				route := ctx.Route()
				tp := reflect.ValueOf(route.Raw()).Elem()
				if !before.Before(tp.Type().Name(),
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
			tp := reflect.ValueOf(route.Raw()).Elem()
			if !after.After(
				tp.Type().Name(),
				route.Method().Type().Name(),
				ctx.Result) {
				fmt.Println("we current cannot disallow invoke to next interceptors")
			}
		}
	}
}
