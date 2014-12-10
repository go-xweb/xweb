package tests

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/go-xweb/xweb"
)

type Cfg struct {
	*xweb.Configs
}

func (cfg *Cfg) SetConfigs(cfgs *xweb.Configs) {
	cfg.Configs = cfgs
}

func (cfg *Cfg) Do() string {
	return "hello " + cfg.GetConfig("hello").(string)
}

func TestCfg(t *testing.T) {
	xweb.MainServer().Config.EnableGzip = false
	xweb.AddRouter("/", new(Hello))
	xweb.RootApp().SetConfig("hello", "xweb")
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
