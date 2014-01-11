// Package web is a lightweight web framework for Go. It's ideal for
// writing simple, performant backend web services.
package xweb

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	//"os"
	"path"
	//"reflect"
	//"strings"
	"fmt"
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
	errorTmpl = `<html><title>%d - %s</title>
	<style>
	body {
		border: 0;
		margin: 0;
		padding: 0;
	}
	</style>
	<body>
	<div style="padding:10px;background-color:#81DAF5;border-bottom: 1px solid #0404B4" width="100%%">
	<h1>%d - %s</h1>
	</div>
	<div style="padding:10px;min-height:500px;">
	%s</div>
	<div style="padding:10px;background-color:#81DAF5;border-top: 1px solid #0404B4" width="100%%">
	xweb v%s <input type="button" style="" value="返回" onclick="history.go(-1)"/>
	</div></body>
	</html>
	`
)

func Error(w http.ResponseWriter, status int, content string) error {
	w.WriteHeader(status)
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
