package tests

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/go-xweb/httpsession"
	"github.com/go-xweb/xweb"
)

type InjectAction struct {
	session  *httpsession.Session
	request  *http.Request
	response *xweb.ResponseWriter
	callthis string
}

func (a *InjectAction) SetSessions(session *httpsession.Session) {
	a.callthis = "call this"
	a.session = session
}

func (a *InjectAction) SetRequest(request *http.Request) {
	fmt.Println("call setrequest")
	a.request = request
	fmt.Println("referer:", request.Referer())
}

func (a *InjectAction) SetResponse(response *xweb.ResponseWriter) {
	fmt.Println("call setresponse")
	a.response = response
	fmt.Println("statuscode:", response.StatusCode)
}

func (a *InjectAction) Execute() string {
	return a.callthis
}

func TestInject(t *testing.T) {
	xweb.MainServer().Config.EnableGzip = false
	xweb.AddAction(new(InjectAction))
	go func() {
		xweb.Run("0.0.0.0:9995")
	}()

	resp, err := http.Get("http://localhost:9995/")
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

	if string(bs) != "call this" {
		t.Error("content is not equeal")
		return
	}
}
