package main

import (
	"fmt"
	"log"
	"os"

	"github.com/lunny/xorm"
	"github.com/lunny/xweb"
	_ "github.com/mattn/go-sqlite3"
)

type MainAction struct {
	*xweb.Action

	root   xweb.Mapper `xweb:"GET /"`
	list   xweb.Mapper `xweb:"GET /list"`
	login  xweb.Mapper
	logout xweb.Mapper
	add    xweb.Mapper `xweb:"GET|POST /add"`
	del    xweb.Mapper `xweb:"GET /delete"`
	edit   xweb.Mapper `xweb:"/edit"`

	Id   int64
	User User
}

type User struct {
	Id     int64
	Name   string
	Passwd string
}

var (
	engine *xorm.Engine
)

var su *User = &User{}

func init() {
	var err error
	engine, err = xorm.NewEngine("sqlite3", "./data.db")
	if err != nil {
		fmt.Println(err)
		return
	}
	engine.ShowSQL = true
	err = engine.CreateTables(su)
	if err != nil {
		fmt.Println(err)
	} else {
		cnt, err := engine.Count(su)
		if err != nil {
			fmt.Println(err)
		} else if cnt == 0 {
			user := User{1, "admin", "123"}
			_, err := engine.Insert(&user)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("Init db successfully!")
			}
		}
	}
}

func (c *MainAction) Root() {
	c.Go("login")
}

func (c *MainAction) IsLogin() bool {
	s := c.GetSession("userId")
	return s != nil
}

func (c *MainAction) Logout() {
	c.DelSession("userId")
	c.Go("root")
}

func (c *MainAction) List() error {
	users := make([]User, 0)
	err := engine.Find(&users)
	if err == nil {
		err = c.Render("list.html", &xweb.T{
			"Users": &users,
		})
	}
	return err
}

func (c *MainAction) Login() {
	if c.Method() == "GET" {
		c.Render("login.html")
	} else if c.Method() == "POST" {
		if c.User.Name == "" || c.User.Passwd == "" {
			c.Go("login")
			return
		}
		has, err := engine.Get(&c.User)
		if err == nil && has {
			if has {
				c.SetSession("userId", c.User.Id)
				c.Go("list")
			} else {
				c.Write("No user %v or password is error", c.User.Name)
			}
		} else {
			c.Write("Login error: %v", err)
		}
	}
}

func (c *MainAction) Add() {
	if c.Method() == "GET" {
		c.Render("add.html")
	} else if c.Method() == "POST" {
		_, err := engine.Insert(&c.User)
		if err == nil {
			c.Go("list")
		} else {
			c.Write("add user error: %v", err)
		}
	}
}

func (c *MainAction) Del() {
	_, err := engine.Id(c.Id).Delete(su)
	if err != nil {
		c.Write("删除失败：%v", err)
	} else {
		c.Go("list")
	}
}

func (c *MainAction) Edit() {
	if c.Method() == "GET" {
		if c.Id > 0 {
			has, err := engine.Id(c.Id).Get(&c.User)
			if err == nil {
				if has {
					c.Render("edit.html")
				} else {
					c.NotFound("no exist")
				}
			} else {
				c.Write("error: %v", err)
			}
		} else {
			c.Write("error: no user id")
		}
	} else if c.Method() == "POST" {
		if c.User.Id > 0 {
			_, err := engine.Id(c.User.Id).Update(&c.User)
			if err == nil {
				c.Go("list")
			} else {
				c.Write("save user error: %v", err)
			}
		} else {
			c.Write("error: no user id")
		}
	}
}

func main() {
	xweb.AddAction(&MainAction{})

	app := xweb.MainServer().RootApp
	filter := xweb.NewLoginFilter(app, "userId", "/login")
	filter.AddAnonymousUrls("/", "/login", "/logout")
	app.AddFilter(filter)
	//app.AppConfig.StaticFileVersion = false

	f, err := os.Create("simple.log")
	if err != nil {
		println(err.Error())
		return
	}
	logger := log.New(f, "", log.Ldate|log.Ltime)
	xweb.SetLogger(logger)
	xweb.Run("0.0.0.0:8080")
}
