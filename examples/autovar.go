package main

import (
	"net/http"

	"github.com/lunny/xweb"
)

type User struct {
	Id      int64
	PtrId   *int64
	Name    string
	PtrName *string
	Age     float32
	PtrAge  *float32
}

type MainAction struct {
	xweb.Action

	get xweb.Mapper `xweb:"/"`

	User  User
	User2 *User
}

func (c *MainAction) Get() {
	c.Write(fmt.Sprintf("%v, %v", c.User, c.User2))
}

func main() {
	xweb.AddAction(&MainAction{})
	go xweb.Run("0.0.0.0:9999")

	http.Get()
}
