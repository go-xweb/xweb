package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/go-xweb/xweb"
)

type User struct {
	Id      int64
	PtrId   *int64
	Name    string
	PtrName *string
	Age     float32
	PtrAge  *float32
	Child   *User
}

type MainAction struct {
	*xweb.Action

	get xweb.Mapper `xweb:"/"`

	User  User
	User2 *User
	Id    *int64
	Key   *string
	Keys  []string
	Keys2 []int
}

func (c *MainAction) Get() {
	right := (*c.Key == "Value" && *c.Id == 123 && c.User.Id == 2 &&
		*c.User2.PtrId == 3 && c.User2.Id == 4 && c.User2.Child.Id == 66)
	if right {
		c.Write(fmt.Sprintf("right!!!\nc.User:%v\nc.User2:%v\nc.User2.PtrId:%v\nc.User2.Child:%v\nc.Id:%v\nc.Key:%v",
			c.User, c.User2, *c.User2.PtrId, c.User2.Child, *c.Id, *c.Key))
	} else {
		c.Write("not right")
	}
	fmt.Println("c.Keys:", c.Keys)
	fmt.Println("c.Keys2:", c.Keys2)
}

func main() {
	xweb.AddAction(&MainAction{})
	xweb.RootApp().AppConfig.CheckXrsf = false
	go xweb.Run("0.0.0.0:9999")

	values := url.Values{"key": {"Value"}, "id": {"123"},
		"user.id": {"2"}, "user2.ptrId": {"3"},
		"user2.id": {"4"}, "user2.child.id": {"66"},
		"keys":  {"1", "2", "3"},
		"keys2": {"1", "2", "3"},
	}
	resp, err := http.PostForm("http://127.0.0.1:9999/", values)
	if err != nil {
		fmt.Println(err)
		return
	}

	bytes, err := ioutil.ReadAll(resp.Body.(io.Reader))
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(bytes))

	var s chan int
	<-s
}
