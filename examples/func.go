package main

import (
	//. "github.com/lunny/xweb"
	"fmt"
	. "xweb"
)

type MainAction struct {
	Action

	hello Mapper `xweb:"/(.*)"`
}

func (c *MainAction) Hello(world string) error {
	return c.RenderString(fmt.Sprintf("hello {{if isWorld}}%v{{else}}go{{end}}", world))
}

func (c *MainAction) IsWorld() bool {
	return true
}

func (c *MainAction) Init() {
	fmt.Println("init mainaction")
	c.AddFunc("isWorld", c.IsWorld)
}

func main() {
	AddRouter("/", &MainAction{})
	Run("0.0.0.0:9999")
}
