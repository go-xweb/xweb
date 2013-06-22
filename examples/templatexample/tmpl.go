package main

import (
	"fmt"
	. "xweb"
)

type MainAction struct {
	Action

	home Mapper `xweb:"/"`
}

func hello1() string {
	return "this hello is in header"
}

func hello2() string {
	return "this hello is in body"
}

func hello3() string {
	return "this hello is in footer"
}

func (this *MainAction) Home() {
	err := this.Render("home.html", &T{
		"title":  "模版测试例子",
		"body":   "模版具体内容",
		"footer": "版权所有",
	}, &T{
		"hello1": hello1,
		"hello2": hello2,
		"hello3": hello3,
	})
	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	AddAction(&MainAction{})
	Run("0.0.0.0:8888")
}
