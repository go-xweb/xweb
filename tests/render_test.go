package tests

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/go-xweb/xweb"
)

type RenderAction struct {
	*xweb.Renderer
}

func (a *RenderAction) SetRenderer(renderer *xweb.Renderer) {
	a.Renderer = renderer
}

func (a *RenderAction) Do() error {
	return a.RenderString("hello {{.T.hello}}", &xweb.T{
		"hello": "world",
	})
}

func TestRender(t *testing.T) {
	xweb.MainServer().Config.EnableGzip = false
	xweb.AddRouter("/", new(RenderAction))
	go func() {
		xweb.Run("0.0.0.0:9992")
	}()

	resp, err := http.Get("http://localhost:9992/")
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

	if string(bs) != "hello world" {
		t.Error("content is not equeal")
		return
	}
}
