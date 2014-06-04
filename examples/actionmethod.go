package main

import (
	"fmt"

	"github.com/go-xweb/xweb"
)

type MainAction struct {
	*xweb.Action

	hello xweb.Mapper `xweb:"/(.*)"`
}

var content string = `
	base path is {{.Basepath}}
`

func (c *MainAction) Basepath() string {
	return c.App.BasePath
}

func (c *MainAction) Hello(world string) {
	err := c.RenderString(content)
	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	xweb.AddAction(&MainAction{})
	xweb.Run("0.0.0.0:9999")
}
