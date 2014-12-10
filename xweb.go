// Package web is a lightweight web framework for Go. It's ideal for
// writing simple, performant backend web services.
package xweb

import (
	"crypto/tls"
	"net/http"

	"github.com/go-xweb/log"
)

const (
	Version = "0.3.0 alpha"
)

// Process invokes the main server's routing system.
func Process(c http.ResponseWriter, req *http.Request) {
	mainServer.Process(c, req)
}

// Run starts the web application and serves HTTP requests for the main server.
func Run(addr string) {
	mainServer.Run(addr)
}

func SimpleTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	config := &tls.Config{}
	if config.NextProtos == nil {
		config.NextProtos = []string{"http/1.1"}
	}

	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return config, nil
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

func AddTmplVar(name string, varOrFun interface{}) {
	mainServer.AddTmplVar(name, varOrFun)
}

func AddTmplVars(t *T) {
	mainServer.AddTmplVars(t)
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

func AddHook(name string, fns ...interface{}) {
	XHook.Bind(name, fns...)
}

func SetTemplateDir(dir string) {
	mainServer.SetTemplateDir(dir)
}

func SetStaticDir(dir string) {
	mainServer.SetStaticDir(dir)
}

// SetLogger sets the logger for the main server.
func SetLogger(logger *log.Logger) {
	mainServer.SetLogger(logger)
}

func MainServer() *Server {
	return mainServer
}

func RootApp() *App {
	return mainServer.RootApp
}

func Serv(name string) *Server {
	server, ok := Servers[name]
	if ok {
		return server
	}
	return nil
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
