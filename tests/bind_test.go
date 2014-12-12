package tests

import (
	"fmt"
	"testing"

	"github.com/go-xweb/xweb"
)

type BindAction struct {
	*xweb.Action

	Id   int64
	Name string
}

func (a *BindAction) Do() string {
	return fmt.Sprintf("%d-%s", a.Id, a.Name)
}

func TestBind(t *testing.T) {
	go func() {
		x := xweb.Classic()
		x.Use(&xweb.Binds{})
		x.AddAction(new(BindAction))
		x.Run("0.0.0.0:9997")
	}()

	res, err := get("http://localhost:9997/?id=1&name=lllll")
	if err != nil {
		t.Error(err)
		return
	}

	if res != "1-lllll" {
		t.Error("not equal")
		return
	}
}
