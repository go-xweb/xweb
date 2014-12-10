package tests

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/go-xweb/xweb"
)

type SessionAction struct {
	*xweb.Action
}

func (a *SessionAction) Do() string {
	a.SetSession("test", "1")
	fmt.Printf("this---->%q\n", *a)
	fmt.Println("test--->", a.GetSession("test"))
	fmt.Println("contentlength:", a.Request.ContentLength)
	fmt.Println("...", a.ResponseWriter.StatusCode)
	return "test"
}

func TestSession(t *testing.T) {
	xweb.MainServer().Config.EnableGzip = false
	xweb.AddAction(new(SessionAction))
	go func() {
		xweb.Run("0.0.0.0:9994")
	}()

	resp, err := http.Get("http://localhost:9994/")
	if err != nil {
		t.Error(err)
		return
	}
	defer resp.Body.Close()

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println("----->", string(bs))

	if string(bs) != "test" {
		t.Error("content is not equeal")
		return
	}
}
