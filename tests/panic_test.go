package tests

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/go-xweb/xweb"
)

type PanicAction struct {
}

func (a *PanicAction) Do() {
	panic("tttttt")
}

func TestPanic(t *testing.T) {
	xweb.AddAction(new(PanicAction))
	go func() {
		xweb.Run(":9993")
	}()
	time.Sleep(time.Second)

	resp, err := http.Get("http://localhost:9993/")
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

	fmt.Println("panic----->", string(bs))
}
