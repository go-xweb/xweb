package tests

import (
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/go-xweb/xweb"
)

var content = "test"

func TestStatic(t *testing.T) {
	os.RemoveAll("./static/")
	os.MkdirAll("./static/", os.ModePerm)
	f, err := os.Create("./static/a.html")
	if err != nil {
		t.Error(err)
		return
	}

	f.Write([]byte(content))
	f.Close()

	go func() {
		xweb.Run("0.0.0.0:9999")
	}()

	resp, err := http.Get("http://localhost:9999/a.html")
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
	//fmt.Println("a.html resp body:", string(bs))

	if string(bs) != content {
		t.Error("content is not equeal")
		return
	}

	resp, err = http.Get("http://localhost:9999/b.html")
	if err != nil {
		t.Error(err)
		return
	}
	defer resp.Body.Close()

	bs, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
		return
	}
	//fmt.Println("b.html resp body:", string(bs))

	if string(bs) == content {
		t.Error("content is equeal")
		return
	}

	os.MkdirAll("./static/test/", os.ModePerm)
	f, err = os.Create("./static/test/index.html")
	if err != nil {
		t.Error(err)
		return
	}

	f.Write([]byte(content))
	f.Close()

	resp, err = http.Get("http://localhost:9999/test/")
	if err != nil {
		t.Error(err)
		return
	}
	defer resp.Body.Close()

	bs, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
		return
	}

	if string(bs) != content {
		t.Error("content is not equeal")
		return
	}
}
