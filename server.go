package xweb

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	runtimePprof "runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/go-xweb/httpsession"
	"github.com/go-xweb/log"
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
	SessionTimeout         time.Duration
}

var ServerNumber uint = 0

// Server represents a xweb server.
type Server struct {
	Config         *ServerConfig
	Apps           map[string]*App
	AppsNamePath   map[string]string
	Name           string
	SessionManager *httpsession.Manager
	RootApp        *App
	Logger         *log.Logger
	Env            map[string]interface{}
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
		Config:       Config,
		Env:          map[string]interface{}{},
		Apps:         map[string]*App{},
		AppsNamePath: map[string]string{},
		Name:         name,
	}
	Servers[s.Name] = s

	s.SetLogger(log.New(os.Stdout, "", log.Ldefault()))

	app := NewApp("/", "root")
	s.AddApp(app)
	return s
}

func (s *Server) AddApp(a *App) {
	a.BasePath = strings.TrimRight(a.BasePath, "/") + "/"
	s.Apps[a.BasePath] = a

	if a.Name != "" {
		s.AppsNamePath[a.Name] = a.BasePath
	}

	a.Server = s
	//a.Logger = s.Logger
	if a.BasePath == "/" {
		s.RootApp = a
	}
}

func (server *Server) Classic() *App {
	app := &App{
		Injector:     NewInjector(),
		Router:       NewRouter("/"),
		interceptors: make([]Interceptor, 0),
		Render: NewRender(
			"templates",
			true,
			true,
		),
	}

	app.Map(server.Logger)

	app.Use(
		NewLogInterceptor(server.Logger),
		NewPanics(
			server.Config.RecoverPanic,
			app.AppConfig.Mode == Debug,
		),
		app.Render,
		NewCompress(server.Config.StaticExtensionsToGzip),
		&ReturnInterceptor{},
		&Static{
			RootPath: "static",
			IndexFiles: []string{
				"index.html",
				"index.htm",
			},
		},
		&Events{},
		&Requests{},
		&Responses{},
		app,
		NewXsrf(time.Minute*20),
		NewSessions(nil, time.Minute*20),
	)

	app.InjectAll()

	return app
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
	s.RootApp.SetConfig(name, value)
}

func (s *Server) SetConfig(name string, value interface{}) {
	s.RootApp.SetConfig(name, value)
}

func (s *Server) GetConfig(name string) interface{} {
	return s.RootApp.GetConfig(name)
}

/*
func (s *Server) error(w http.ResponseWriter, status int, content string) error {
	return s.RootApp.error(w, status, content)
}*/

func (s *Server) initServer() {
	if s.Config == nil {
		s.Config = &ServerConfig{}
		s.Config.Profiler = true
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
	var result bool = true
	_, _ = XHook.Call("BeforeProcess", &result, s, w, req)
	if !result {
		return
	}
	if s.Config.UrlSuffix != "" && strings.HasSuffix(req.URL.Path, s.Config.UrlSuffix) {
		req.URL.Path = strings.TrimSuffix(req.URL.Path, s.Config.UrlSuffix)
	}
	if s.Config.UrlPrefix != "" && strings.HasPrefix(req.URL.Path, "/"+s.Config.UrlPrefix) {
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/"+s.Config.UrlPrefix)
	}
	if req.URL.Path[0] != '/' {
		req.URL.Path = "/" + req.URL.Path
	}
	for _, app := range s.Apps {
		if app != s.RootApp && strings.HasPrefix(req.URL.Path, app.BasePath) {
			app.ServeHttp(w, req)
			return
		}
	}
	s.RootApp.ServeHttp(w, req)
	_, _ = XHook.Call("AfterProcess", &result, s, w, req)
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
		mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		mux.Handle("/debug/pprof/block", pprof.Handler("block"))
		mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))

		mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))

		mux.Handle("/debug/pprof/startcpuprof", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			StartCPUProfile()
		}))
		mux.Handle("/debug/pprof/stopcpuprof", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			StopCPUProfile()
		}))
		mux.Handle("/debug/pprof/memprof", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			runtime.GC()
			runtimePprof.WriteHeapProfile(rw)
		}))
		mux.Handle("/debug/pprof/gc", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			PrintGCSummary(rw)
		}))

	}

	if c, err := XHook.Call("MuxHandle", mux); err == nil {
		if ret := XHook.Value(c, 0); ret != nil {
			mux = ret.(*http.ServeMux)
		}
	}
	mux.Handle("/", s)

	s.Logger.Infof("http server is listening %s", addr)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		s.Logger.Error("ListenAndServe:", err)
	}
	s.l = l
	err = http.Serve(s.l, mux)
	s.l.Close()
}

// RunFcgi starts the web application and serves FastCGI requests for s.
func (s *Server) RunFcgi(addr string) {
	s.initServer()
	s.Logger.Infof("fcgi server is listening %s", addr)
	s.listenAndServeFcgi(addr)
}

// RunScgi starts the web application and serves SCGI requests for s.
func (s *Server) RunScgi(addr string) {
	s.initServer()
	s.Logger.Infof("scgi server is listening %s", addr)
	s.listenAndServeScgi(addr)
}

// RunTLS starts the web application and serves HTTPS requests for s.
func (s *Server) RunTLS(addr string, config *tls.Config) error {
	s.initServer()
	mux := http.NewServeMux()
	mux.Handle("/", s)
	l, err := tls.Listen("tcp", addr, config)
	if err != nil {
		s.Logger.Errorf("Listen: %v", err)
		return err
	}

	s.l = l

	s.Logger.Infof("https server is listening %s", addr)

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
	/*if s.RootApp != nil {
		s.RootApp.Logger = s.Logger
	}*/
}

func (s *Server) InitSession() {
	if s.SessionManager == nil {
		s.SessionManager = httpsession.Default()
	}
	if s.Config.SessionTimeout > time.Second {
		s.SessionManager.SetMaxAge(s.Config.SessionTimeout)
	}
	s.SessionManager.Run()
	if s.RootApp != nil {
		s.RootApp.SessionManager = s.SessionManager
	}
}

func (s *Server) SetTemplateDir(path string) {
	s.RootApp.SetTemplateDir(path)
}

func (s *Server) SetStaticDir(path string) {
	s.RootApp.SetStaticDir(path)
}

func (s *Server) App(name string) *App {
	path, ok := s.AppsNamePath[name]
	if ok {
		return s.Apps[path]
	}
	return nil
}
