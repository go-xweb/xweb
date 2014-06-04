package main

import (
	"fmt"
	"html"

	"github.com/go-xweb/xweb"
)

var cookieName = "cookie"

var notice = `
<div>%v</div>
`
var form = `
<form method="POST" action="update">
  {{XsrfFormHtml}}
  <div class="field">
    <label for="cookie"> Set a cookie: </label>
    <input id="cookie" name="cookie"> </input>
  </div>

  <input type="submit" value="Submit"></input>
  <input type="submit" name="submit" value="Delete"></input>
</form>
`

type CookieAction struct {
	*xweb.Action

	index  xweb.Mapper `xweb:"/"`
	update xweb.Mapper
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
		this.SetCookie(xweb.NewCookie(cookieName, "", -1))
	} else {
		this.SetCookie(xweb.NewCookie(cookieName, this.GetString("cookie"), 0))
	}
	this.Redirect("/", 301)
}

func main() {
	xweb.AddAction(&CookieAction{})
	xweb.Run("0.0.0.0:9999")
}
