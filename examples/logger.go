package main

import (
	. "github.com/lunny/xweb"
	"log"
	"os"
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
	f, err := os.Create("server.log")
	if err != nil {
		println(err.Error())
		return
	}
	logger := log.New(f, "", log.Ldate|log.Ltime)

	AddAction(&MainAction{})
	SetLogger(logger)
	Run("0.0.0.0:9999")
}
