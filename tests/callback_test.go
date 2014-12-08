package tests

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/go-xweb/xweb"
)

type TestAction struct {
	*xweb.Action
}

func (a *TestAction) Execute() string {
	return "sssss"
}

var outputsEq = []string{
	"------> init",
	"------> before TestAction Execute",
	"------> after TestAction Execute",
}

var outputs = make([]string, 0)

func (*TestAction) Init() {
	fmt.Println("------> init")
	outputs = append(outputs, "------> init")
}

func (*TestAction) Before(structName, actionName string) bool {
	fmt.Println("------> before", structName, actionName)
	outputs = append(outputs, fmt.Sprintf("------> before %s %s", structName, actionName))
	return true
}

func (*TestAction) After(structName, actionName string, result interface{}) bool {
	fmt.Println("------> after", structName, actionName)
	outputs = append(outputs, fmt.Sprintf("------> after %s %s", structName, actionName))
	return true
}

func TestCallback(t *testing.T) {
	xweb.MainServer().Config.EnableGzip = false
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

	if strings.Join(outputsEq, "") != strings.Join(outputs, "") {
		t.Error("should equal", outputsEq, outputs)
	}
}
