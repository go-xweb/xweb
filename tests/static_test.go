package tests

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/go-xweb/xweb"
)

var content = "test"

func gzipDecode(src []byte) ([]byte, error) {
	rd := bytes.NewReader(src)
	b, err := gzip.NewReader(rd)
	if err != nil {
		return nil, err
	}

	defer b.Close()

	data, err := ioutil.ReadAll(b)
	if err != nil {
		return nil, err
	}

	return data, nil
}

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

	var output string
	if xweb.MainServer().Config.EnableGzip {
		data, err := gzipDecode(bs)
		if err != nil {
			t.Error(err)
			return
		}
		output = string(data)
	} else {
		output = string(bs)
	}

	if output != content {
		t.Error("content is not equeal", output)
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

	if xweb.MainServer().Config.EnableGzip {
		data, err := gzipDecode(bs)
		if err != nil {
			t.Error(err)
			return
		}
		output = string(data)
	} else {
		output = string(bs)
	}

	if output == content {
		t.Error("content is equeal", string(bs), output)
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

	if xweb.MainServer().Config.EnableGzip {
		data, err := gzipDecode(bs)
		if err != nil {
			t.Error(err)
			return
		}
		output = string(data)
	} else {
		output = string(bs)
	}

	if output != content {
		t.Error("content is not equeal", string(bs), output)
		return
	}
}
