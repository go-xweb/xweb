package main

import (
	"github.com/lunny/xweb"
)

var page = `
<html>
<head><title>Multipart Test</title></head>
<body>
<form action="/" method="POST">
<input type="text" name="user.id"/>
<input type="text" name="user.name"/>
<input type="text" name="user.age"/>
<input type="submit" name="Submit" value="Submit"/>
</form>
</body>
</html>
`

type MainAction struct {
	*xweb.Action

	parse xweb.Mapper `xweb:"/"`
}

type User struct {
	Id   int64
	Name string
	Age  float64
}

func (c *MainAction) Init() {
	c.Option.AutoMapForm = false
	c.Option.CheckXrsf = false
}

func (c *MainAction) Parse() error {
	if c.Method() == "GET" {
		return c.Write(page)
	} else if c.Method() == "POST" {
		var user User
		err := c.MapForm(&user)
		if err != nil {
			return err
		}
		return c.Write("%v", user)
	}
	return nil
}

func main() {
	xweb.AddAction(&MainAction{})
	xweb.Run("0.0.0.0:9999")
}
