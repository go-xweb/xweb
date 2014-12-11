package tests

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/go-xweb/xweb"
)

func TestClassic(t *testing.T) {
	go func() {
		x := xweb.Classic()
		x.AddRouter("/", new(Hello))
		x.Run("0.0.0.0:9990")
	}()

	resp, err := http.Get("http://localhost:9990/")
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
