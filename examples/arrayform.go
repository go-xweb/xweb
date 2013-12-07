package main

import (
	"github.com/lunny/xweb"
)

var page = `
<html>
<head><title>Multipart Test</title></head>
<body>
<form action="/" method="POST">
<label for="input1"> Please write some text </label>
<input id="input1" type="text" name="inputs"/>
<br>
<label for="input2"> Please write some more text </label>
<input id="input2" type="text" name="inputs"/>
<br>
<input type="submit" name="Submit" value="Submit"/>
</form>
</body>
</html>
`

type MainAction struct {
	xweb.Action

	upload xweb.Mapper `xweb:"/"`

	Inputs []string
}

func (c *MainAction) Upload() {
	if c.Method() == "GET" {
		c.Write(page)
	} else if c.Method() == "POST" {
		for i, input := range c.Inputs {
			c.Write("<p>input %v: %v </p>", i, input)
		}
	}
}

func main() {
	xweb.AddAction(&MainAction{})
	xweb.Run("0.0.0.0:9999")
}
