package main

import (
	"fmt"

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
	xweb.RootApp().AppConfig.SessionOn = false
	xweb.AddRouter("/", &MainAction{})

	config, err := xweb.SimpleTLSConfig("cert.pem", "key.pem")
	if err != nil {
		fmt.Println(err)
		return
	}

	xweb.RunTLS("0.0.0.0:9999", config)
}
