package main

import (
	"github.com/go-xweb/xweb"
)

type MainAction struct {
	*xweb.Action

	hello xweb.Mapper `xweb:"/(.*)"`
}

func (c *MainAction) Hello(world string) {
	c.Write("hello %v", world)
}

func main() {
	xweb.RootApp().AppConfig.SessionOn = false
	xweb.AddRouter("/", &MainAction{})
	xweb.Run("0.0.0.0:9999")
}
