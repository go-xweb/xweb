package main

import (
	"fmt"
	"time"

	"github.com/go-xweb/xweb"
)

type MainAction struct {
	*xweb.Action

	start time.Time

	hello xweb.Mapper `xweb:"/(.*)"`
}

func (c *MainAction) Hello(world string) {
	c.Write("hello %v", world)
}

func (c *MainAction) Before(structName, actionName string) bool {
	c.start = time.Now()
	fmt.Println("before", c.start)
	return true
}

func (c *MainAction) After(structName, actionName string, actionResult interface{}) bool {
	fmt.Println("after", time.Now().Sub(c.start))
	return true
}

func main() {
	xweb.AddRouter("/", &MainAction{})
	xweb.Run("0.0.0.0:9999")
}
