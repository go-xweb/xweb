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
    mc := &MainAction{}

    server1 := xweb.NewServer()
    server1.AddRouter("/", mc)
    go server1.Run("0.0.0.0:9999")

    server2 := xweb.NewServer()
    server2.AddRouter("/", mc)
    go server2.Run("0.0.0.0:8999")

    <-make(chan int)
}
