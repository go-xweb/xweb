// Package web is a lightweight web framework for Go. It's ideal for
// writing simple, performant backend web services.
package xweb

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	//"reflect"
	"fmt"
	//"strings"
)

const (
	version = "0.1.2"
)

// small optimization: cache the context type instead of repeteadly calling reflect.Typeof
//var contextType reflect.Type

func init() {
	//contextType = reflect.TypeOf(Context{})
	//find the location of the exe file
	/*wd, _ := os.Getwd()
	arg0 := path.Clean(os.Args[0])
	var exeFile string
	if strings.HasPrefix(arg0, "/") {
		exeFile = arg0
	} else {
		//TODO for robustness, search each directory in $PATH
		exeFile = path.Join(wd, arg0)
	}
	_, _ := path.Split(exeFile)*/
	return
}

func Redirect(w http.ResponseWriter, url string) error {
	w.Header().Set("Location", url)
	w.WriteHeader(302)
	_, err := w.Write([]byte("Redirecting to: " + url))
	return err
}

func Download(w http.ResponseWriter, fpath string) error {
	data, err := ioutil.ReadFile(fpath)
	if err != nil {
		return err
	}
	fName := fpath[len(path.Dir(fpath))+1:]
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%v\"", fName))
	_, err = w.Write(data)
	return err
}

const (
	defaultErrorTmpl        = `<!DOCTYPE html>
<html lang="en">
	<meta charset="UTF-8" />
	<title>%d - %s</title>
	<style type="text/css">
	body{border:0;margin:0;padding:0;background:#eee url('data:image/gif;base64,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') right bottom;}
	.error{width:600px;border:1px solid #ccc;margin:100px auto;border-radius:5px;box-shadow:0 0 10px #ccc;background:#fff url('data:image/gif;base64,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') no-repeat right bottom;}
	.error-body,.error-foot{margin:0 10px 10px 10px;}
	.error-body{margin-top:10px;min-height:180px}
	.error-head h1{padding:10px 10px 0 10px;margin:0 2px;vertical-align:middle;border-bottom:1px solid #eee;}
	.framework{color:#eee;font-size:11px}
	</style>
	<body>
		<div class="error">
		<div class="error-head">
			<h1>%d - %s</h1>
		</div>
		<div class="error-body">%s</div>
		<div class="error-foot">
			<input type="button" title="Back" value="返回" onclick="history.go(-1)"/>
			<em class="framework">(xweb v%s)</em>
		</div>
		</div>
	</body>
</html>`
)
var errorTmpl string = ""

func Error(w http.ResponseWriter, status int, content string) error {
	w.WriteHeader(status)
	if errorTmpl == "" {
		errTmplFile := RootApp().AppConfig.TemplateDir + "/_error.html"
		if file, err := os.Stat(errTmplFile); err == nil && !file.IsDir() {
			if b, e := ioutil.ReadFile(errTmplFile); e == nil {
				errorTmpl = string(b)
			}
		}
		if errorTmpl == "" {
			errorTmpl = defaultErrorTmpl
		}
	}
	res := fmt.Sprintf(errorTmpl, status, statusText[status],
		status, statusText[status], content, version)
	_, err := w.Write([]byte(res))
	return err
}

// Process invokes the main server's routing system.
func Process(c http.ResponseWriter, req *http.Request) {
	mainServer.Process(c, req)
}

// Run starts the web application and serves HTTP requests for the main server.
func Run(addr string) {
	mainServer.Run(addr)
}

// RunTLS starts the web application and serves HTTPS requests for the main server.
func RunTLS(addr string, config *tls.Config) {
	mainServer.RunTLS(addr, config)
}

// RunScgi starts the web application and serves SCGI requests for the main server.
func RunScgi(addr string) {
	mainServer.RunScgi(addr)
}

// RunFcgi starts the web application and serves FastCGI requests for the main server.
func RunFcgi(addr string) {
	mainServer.RunFcgi(addr)
}

// Close stops the main server.
func Close() {
	mainServer.Close()
}

func AutoAction(c ...interface{}) {
	mainServer.AutoAction(c...)
}

func AddAction(c ...interface{}) {
	mainServer.AddAction(c...)
}

func AddRouter(url string, c interface{}) {
	mainServer.AddRouter(url, c)
}

func AddFilter(filter Filter) {
	mainServer.AddFilter(filter)
}

func AddApp(a *App) {
	mainServer.AddApp(a)
}

func AddConfig(name string, value interface{}) {
	mainServer.AddConfig(name, value)
}

func SetTemplateDir(dir string) {
	mainServer.SetTemplateDir(dir)
}

func SetStaticDir(dir string) {
	mainServer.SetStaticDir(dir)
}

// SetLogger sets the logger for the main server.
func SetLogger(logger *log.Logger) {
	mainServer.Logger = logger
}

func MainServer() *Server {
	return mainServer
}

func RootApp() *App {
	return MainServer().RootApp
}

// Config is the configuration of the main server.
var (
	Config *ServerConfig = &ServerConfig{
		RecoverPanic: true,
		EnableGzip:   true,
		//Profiler: true,
		StaticExtensionsToGzip: []string{".css", ".js"},
	}
	Servers    map[string]*Server = make(map[string]*Server) //[SWH|+]
	mainServer *Server            = NewServer("main")
)
