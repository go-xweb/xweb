package main

import (
	"os"

	"github.com/go-xweb/log"
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
	f, err := os.Create("server.log")
	if err != nil {
		println(err.Error())
		return
	}
	logger := log.New(f, "", log.Ldate|log.Ltime)

	xweb.AddAction(&MainAction{})
	xweb.SetLogger(logger)
	xweb.Run("0.0.0.0:9999")
}
