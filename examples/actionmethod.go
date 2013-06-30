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
	AddAction(&MainAction{})
	Run("0.0.0.0:9999")
}
