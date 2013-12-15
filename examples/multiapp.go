package main

import (
    "github.com/lunny/xweb"
)

type MainAction struct {
    xweb.Action

    hello xweb.Mapper `xweb:"/(.*)"`
}

func (c *MainAction) Hello(world string) {
    c.Write("hello %v", world)
}

func main() {
    app1 := xweb.NewApp("/")
    app1.AddAction(&MainAction{})
    xweb.AddApp(app1)

    app2 := xweb.NewApp("/user/")
    app2.AddAction(&MainAction{})
    xweb.AddApp(app2)

    xweb.Run("0.0.0.0:9999")
}
