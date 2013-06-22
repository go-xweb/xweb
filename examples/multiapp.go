package main

import (
	. "github.com/lunny/xweb"
	//. "xweb"
)

type MainAction struct {
	Action

	hello Mapper `xweb:"/(.*)"`
}

func (c *MainAction) Hello(world string) {
	c.Write("hello %v", world)
}

func main() {
	app1 := NewApp("/")
	app1.AddAction(&MainAction{})
	AddApp(app1)

	app2 := NewApp("/user/")
	app2.AddAction(&MainAction{})
	AddApp(app2)

	Run("0.0.0.0:9999")
}
