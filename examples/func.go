package main

import (
	"fmt"

	"github.com/go-xweb/xweb"
)

type MainAction struct {
	*xweb.Action

	hello xweb.Mapper `xweb:"/(.*)"`
}

func (c *MainAction) Hello(world string) error {
	return c.RenderString(fmt.Sprintf("hello {{if isWorld}}%v{{else}}go{{end}}", world))
}

func (c *MainAction) IsWorld() bool {
	return true
}

func (c *MainAction) Init() {
	fmt.Println("init mainaction")
	c.AddTmplVar("isWorld", c.IsWorld)
}

func main() {
	xweb.AddAction(&MainAction{})
	xweb.Run("0.0.0.0:9999")
}
