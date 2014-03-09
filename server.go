package xweb

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// ServerConfig is configuration for server objects.
type ServerConfig struct {
	Addr                   string
	Port                   int
	RecoverPanic           bool
	Profiler               bool
	EnableGzip             bool
	StaticExtensionsToGzip []string
	Url                    string
	UrlPrefix              string
	UrlSuffix              string
}

var ServerNumber uint = 0

// Server represents a xweb server.
type Server struct {
	Config   *ServerConfig
	Apps     map[string]*App
	AppName  map[string]string //[SWH|+]
	Name     string            //[SWH|+]
	RootApp  *App
	Logger   *log.Logger
	osLogger osLogger
	Env      map[string]interface{}
	//save the listener so it can be closed
	l net.Listener
}

func NewServer(args ...string) *Server {
	name := ""
	if len(args) == 1 {
		name = args[0]
	} else {
		name = fmt.Sprintf("Server%d", ServerNumber)
		ServerNumber++
	}
	s := &Server{
		Config:  Config,
		Logger:  log.New(os.Stdout, "", log.Ldate|log.Ltime),
		Env:     map[string]interface{}{},
		Apps:    map[string]*App{},
		AppName: map[string]string{},
		Name:    name,
	}
	Servers[s.Name] = s //[SWH|+]

	s.SetLogger(log.New(os.Stdout, "", log.Ldate|log.Ltime))

	app := NewApp("/", "root") //[SWH|+] ,"root"
	s.AddApp(app)
	return s
}

func (s *Server) AddApp(a *App) {
	a.BasePath = strings.TrimRight(a.BasePath, "/") + "/"
	s.Apps[a.BasePath] = a

	//[SWH|+]
	if a.Name != "" {
		s.AppName[a.Name] = a.BasePath
	}

	a.Server = s
	a.Logger = s.Logger
	if a.BasePath == "/" {
		s.RootApp = a
	}
}

func (s *Server) AddAction(cs ...interface{}) {
	s.RootApp.AddAction(cs...)
}

func (s *Server) AutoAction(c ...interface{}) {
	s.RootApp.AutoAction(c...)
}

func (s *Server) AddRouter(url string, c interface{}) {
	s.RootApp.AddRouter(url, c)
}

func (s *Server) AddTmplVar(name string, varOrFun interface{}) {
	s.RootApp.AddTmplVar(name, varOrFun)
}

func (s *Server) AddTmplVars(t *T) {
	s.RootApp.AddTmplVars(t)
}

func (s *Server) AddFilter(filter Filter) {
	s.RootApp.AddFilter(filter)
}

func (s *Server) AddConfig(name string, value interface{}) {
	s.RootApp.Config[name] = value
}

func (s *Server) error(w http.ResponseWriter, status int, content string) error {
	return s.RootApp.error(w, status, content)
}

func (s *Server) initServer() {
	if s.Config == nil {
		s.Config = &ServerConfig{}
	}

	for _, app := range s.Apps {
		app.initApp()
	}
}

// ServeHTTP is the interface method for Go's http server package
func (s *Server) ServeHTTP(c http.ResponseWriter, req *http.Request) {
	s.Process(c, req)
}

// Process invokes the routing system for server s
// non-root app's route will override root app's if there is same path
func (s *Server) Process(w http.ResponseWriter, req *http.Request) {
	for _, app := range s.Apps {
		if app != s.RootApp && strings.HasPrefix(req.URL.Path, app.BasePath) {
			app.routeHandler(req, w)
			return
		}
	}
	s.RootApp.routeHandler(req, w)
}

// Run starts the web application and serves HTTP requests for s
func (s *Server) Run(addr string) {
	addrs := strings.Split(addr, ":")
	s.Config.Addr = addrs[0]
	s.Config.Port, _ = strconv.Atoi(addrs[1])

	s.initServer()

	mux := http.NewServeMux()
	if s.Config.Profiler {
		mux.Handle("/debug/pprof", http.HandlerFunc(pprof.Index))
		mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	}
	//[SWH|+]call hook
	if c, err := XHook.Call("MuxHandle", mux); err == nil {
		if ret := XHook.Value(c, 0); ret != nil {
			mux = ret.(*http.ServeMux)
		}
	}
	mux.Handle("/", s)

	s.Info("http server is listening %s", addr)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		s.Error("ListenAndServe:", err)
	}
	s.l = l
	err = http.Serve(s.l, mux)
	s.l.Close()
}

// RunFcgi starts the web application and serves FastCGI requests for s.
func (s *Server) RunFcgi(addr string) {
	s.initServer()
	s.Info("fcgi server is listening %s", addr)
	s.listenAndServeFcgi(addr)
}

// RunScgi starts the web application and serves SCGI requests for s.
func (s *Server) RunScgi(addr string) {
	s.initServer()
	s.Info("scgi server is listening %s", addr)
	s.listenAndServeScgi(addr)
}

// RunTLS starts the web application and serves HTTPS requests for s.
func (s *Server) RunTLS(addr string, config *tls.Config) error {
	s.initServer()
	mux := http.NewServeMux()
	mux.Handle("/", s)
	l, err := tls.Listen("tcp", addr, config)
	if err != nil {
		s.Error("Listen: %v", err)
		return err
	}

	s.l = l

	s.Info("https server is listening %s", addr)

	return http.Serve(s.l, mux)
}

// Close stops server s.
func (s *Server) Close() {
	if s.l != nil {
		s.l.Close()
	}
}

// SetLogger sets the logger for server s
func (s *Server) SetLogger(logger *log.Logger) {
	s.Logger = logger
	s.Logger.SetPrefix("[" + s.Name + "] ")
	if runtime.GOOS == "windows" {
		s.osLogger = winLog
	} else {
		s.osLogger = unixLog
	}
}

func (s *Server) Trace(format string, params ...interface{}) {
	s.osLogger(s.Logger, ForeCyan, format, params...)
}

func (s *Server) Debug(format string, params ...interface{}) {
	s.osLogger(s.Logger, ForeBlue, format, params...)
}

func (s *Server) Info(format string, params ...interface{}) {
	s.osLogger(s.Logger, ForeGreen, format, params...)
}

func (s *Server) Warn(format string, params ...interface{}) {
	s.osLogger(s.Logger, ForeYellow, format, params...)
}

func (s *Server) Error(format string, params ...interface{}) {
	s.osLogger(s.Logger, ForeRed, format, params...)
}

func (s *Server) Critical(format string, params ...interface{}) {
	s.osLogger(s.Logger, ForePurple, format, params...)
}

func (s *Server) SetTemplateDir(path string) {
	s.RootApp.SetTemplateDir(path)
}

func (s *Server) SetStaticDir(path string) {
	s.RootApp.SetStaticDir(path)
}

func (s *Server) App(name string) *App {
	path, ok := s.AppName[name]
	if ok {
		return s.Apps[path]
	}
	return nil
}
