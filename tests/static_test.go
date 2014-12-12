package tests

import (
	"os"
	"testing"

	"github.com/go-xweb/xweb"
)

var content = "test"



func TestStatic(t *testing.T) {
	os.RemoveAll("./static/")
	os.MkdirAll("./static/", os.ModePerm)
	defer os.RemoveAll("./static/")
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

	output, err := get("http://localhost:9999/a.html")
	if err != nil {
		t.Error(err)
		return
	}

	if output != content {
		t.Error("content is not equeal", output)
		return
	}



	output, err = get("http://localhost:9999/b.html")
	if err != nil {
		t.Error(err)
		return
	}

	if output == content {
		t.Error("content is equeal", output, output)
		return
	}

	os.MkdirAll("./static/test/", os.ModePerm)
	defer os.RemoveAll("./static/tests/")
	f, err = os.Create("./static/test/index.html")
	if err != nil {
		t.Error(err)
		return
	}

	f.Write([]byte(content))
	f.Close()

	output, err = get("http://localhost:9999/test/")
	if err != nil {
		t.Error(err)
		return
	}

	if output != content {
		t.Error("content is not equeal", output, output)
		return
	}
}
