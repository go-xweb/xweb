package main

import (
	//. "github.com/lunny/xweb"
	. "xweb"
)

type MainAction struct {
	Action

	hello Mapper `xweb:"/(.*)"`
}

func (c *MainAction) Hello(world string) {
	c.Write("hello %v", world)
}

func main() {
	AutoAction(&MainAction{})
	Run("0.0.0.0:9999")
	//visit http://localhost:9999/main/world
}
