package tests

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/go-xweb/xweb"
)

func TestXsrf(t *testing.T) {
	xweb.RootApp().AppConfig.CheckXsrf = true
	xweb.AddAction(new(BindAction))
	go func() {
		xweb.Run("0.0.0.0:9996")
	}()

	resp, err := http.Post("http://localhost:9996/?id=1&name=lllll", "", nil)
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

	fmt.Println(string(bs))

	if xweb.RootApp().AppConfig.CheckXsrf {
		if resp.StatusCode == http.StatusOK {
			t.Error("should say xsrf error.")
			return
		}
	} else {
		if resp.StatusCode != http.StatusOK {
			t.Error("should say ok.")
			return
		}
	}
}
