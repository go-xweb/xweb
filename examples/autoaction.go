package main

import (
	"github.com/lunny/xweb"
)

type MainAction struct {
	*xweb.Action

	hello xweb.Mapper `xweb:"/(.*)"`
}

func (c *MainAction) Hello(world string) {
	c.Write("hello %v", world)
}

func main() {
	xweb.AutoAction(&MainAction{})
	xweb.Run("0.0.0.0:9999")
	//visit http://localhost:9999/main/world
}
