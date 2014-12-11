package tests

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/go-xweb/xweb"
)

func TestFunc(t *testing.T) {
	xweb.MainServer().Config.EnableGzip = false
	xweb.AddRouter("/", func() string {
		return "hello xweb"
	})

	go func() {
		xweb.Run("0.0.0.0:9991")
	}()

	resp, err := http.Get("http://localhost:9991/")
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

	fmt.Println("/ resp body:", string(bs))

	if string(bs) != "hello xweb" {
		t.Error("should equal", "hello xweb", string(bs))
	}
}
