package xweb

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"
	"strings"
)

// ServerConfig is configuration for server objects.
type ServerConfig struct {
	Addr         string
	Port         int
	RecoverPanic bool
	Profiler     bool
}

// Server represents a xweb server.
type Server struct {
	Config  *ServerConfig
	Apps    map[string]*App
	RootApp *App
	Logger  *log.Logger
	Env     map[string]interface{}
	//save the listener so it can be closed
	l net.Listener
}

func NewServer() *Server {
	s := &Server{
		Config: Config,
		Logger: log.New(os.Stdout, "", log.Ldate|log.Ltime),
		Env:    map[string]interface{}{},
		Apps:   map[string]*App{},
	}

	app := NewApp("/")
	s.AddApp(app)
	return s
}

func (s *Server) AddApp(a *App) {
	a.BasePath = strings.TrimRight(a.BasePath, "/") + "/"
	s.Apps[a.BasePath] = a
	a.Server = s
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

func (s *Server) AddFilter(filter Filter) {
	s.RootApp.AddFilter(filter)
}

func (s *Server) AddConfig(name string, value interface{}) {
	s.RootApp.Config[name] = value
}

func (s *Server) initServer() {
	if s.Config == nil {
		s.Config = &ServerConfig{}
	}

	if s.Logger == nil {
		s.Logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)
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
func (s *Server) Process(c http.ResponseWriter, req *http.Request) {
	for _, app := range s.Apps {
		if app != s.RootApp && strings.HasPrefix(req.URL.Path, app.BasePath) {
			app.routeHandler(req, c)
			return
		}
	}
	s.RootApp.routeHandler(req, c)
}

// Run starts the web application and serves HTTP requests for s
func (s *Server) Run(addr string) {
	addrs := strings.Split(addr, ":")
	s.Config.Addr = addrs[0]
	s.Config.Port, _ = strconv.Atoi(addrs[1])

	s.initServer()

	mux := http.NewServeMux()
	if s.Config.Profiler {
		mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	}
	mux.Handle("/", s)

	s.Logger.Printf("xweb serving %s\n", addr)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
	s.l = l
	err = http.Serve(s.l, mux)
	s.l.Close()
}

// RunFcgi starts the web application and serves FastCGI requests for s.
func (s *Server) RunFcgi(addr string) {
	s.initServer()
	s.Logger.Printf("xweb serving fcgi %s\n", addr)
	s.listenAndServeFcgi(addr)
}

// RunScgi starts the web application and serves SCGI requests for s.
func (s *Server) RunScgi(addr string) {
	s.initServer()
	s.Logger.Printf("xweb serving scgi %s\n", addr)
	s.listenAndServeScgi(addr)
}

// RunTLS starts the web application and serves HTTPS requests for s.
func (s *Server) RunTLS(addr string, config *tls.Config) error {
	s.initServer()
	mux := http.NewServeMux()
	mux.Handle("/", s)
	l, err := tls.Listen("tcp", addr, config)
	if err != nil {
		log.Fatal("Listen:", err)
		return err
	}

	s.l = l
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
}

func (s *Server) SetTemplateDir(path string) {
	s.RootApp.SetTemplateDir(path)
}

func (s *Server) SetStaticDir(path string) {
	s.RootApp.SetStaticDir(path)
}
