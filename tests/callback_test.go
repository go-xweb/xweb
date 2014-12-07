package tests

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/go-xweb/xweb"
)

type TestAction struct {
	*xweb.Action
}

func (a *TestAction) Execute() string {
	return "sssss"
}

func (*TestAction) Init() {
	fmt.Println("------> init")
}

func (*TestAction) Before(structName, actionName string) bool {
	fmt.Println("------> before", structName, actionName)
	return true
}

func (*TestAction) After(structName, actionName string, result interface{}) bool {
	fmt.Println("------> after", structName, actionName)
	return true
}

func TestCallback(t *testing.T) {
	xweb.AddAction(new(TestAction))
	go func() {
		xweb.Run("0.0.0.0:9998")
	}()

	resp, err := http.Get("http://localhost:9998/")
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

	time.Sleep(time.Minute)
}
