package main

import (
	. "github.com/lunny/xweb"
	//. "xweb"
)

type MainAction struct {
	Action

	hello Mapper `xweb:"/(.*)"`
}

var content string = `
	base path is {{.BasePath}}
`

func (c *MainAction) BasePath() string {
	return c.App.BasePath
}

func (c *MainAction) Hello(world string) {
	c.RenderString(content)
}

func main() {
	AddAction(&MainAction{})
	Run("0.0.0.0:9999")
}
