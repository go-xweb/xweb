package main

import (
	"fmt"
	. "github.com/lunny/xweb"
	"html"
	//. "xweb"
)

var cookieName = "cookie"

var notice = `
<div>%v</div>
`
var form = `
<form method="POST" action="update">
  <div class="field">
    <label for="cookie"> Set a cookie: </label>
    <input id="cookie" name="cookie"> </input>
  </div>

  <input type="submit" value="Submit"></input>
  <input type="submit" name="submit" value="Delete"></input>
</form>
`

type CookieAction struct {
	Action

	index  Mapper `xweb:"/"`
	update Mapper
}

func (this *CookieAction) Index() string {
	cookie, _ := this.GetCookie(cookieName)
	var top string
	if cookie == nil {
		top = fmt.Sprintf(notice, "The cookie has not been set")
	} else {
		var val = html.EscapeString(cookie.Value)
		top = fmt.Sprintf(notice, "The value of the cookie is '"+val+"'.")
	}
	return top + form
}

func (this *CookieAction) Update() {
	if this.GetString("submit") == "Delete" {
		this.SetCookie(NewCookie(cookieName, "", -1))
	} else {
		this.SetCookie(NewCookie(cookieName, this.GetString("cookie"), 0))
	}
	this.Redirect("/", 301)
}

func main() {
	AddAction(&CookieAction{})
	Run("0.0.0.0:9999")
}
