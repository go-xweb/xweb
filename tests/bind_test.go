package tests

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/go-xweb/xweb"
)

type BindAction struct {
	*xweb.Action

	Id   int64
	Name string
}

func (a *BindAction) Execute() string {
	return fmt.Sprintf("%d-%s", a.Id, a.Name)
}

func TestBind(t *testing.T) {
	xweb.MainServer().Config.EnableGzip = false
	xweb.AddAction(new(BindAction))
	go func() {
		xweb.Run("0.0.0.0:9997")
	}()

	resp, err := http.Get("http://localhost:9997/?id=1&name=lllll")
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

	if string(bs) != "1-lllll" {
		t.Error("not equal")
		return
	}
}
