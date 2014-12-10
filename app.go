package xweb

import (
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-xweb/httpsession"
)

type App struct {
	*Injector
	*Router
	*Render
	*Configs

	BasePath string
	Name     string //[SWH|+]

	Server    *Server
	AppConfig AppConfig
	Config    map[string]interface{}

	// TODO: refactoring this
	SessionManager *httpsession.Manager //Session manager

	interceptors []Interceptor
}

const (
	Debug = iota + 1
	Product
)

func NewApp(args ...string) *App {
	basePath := args[0]
	var name string
	if len(args) == 1 {
		name = strings.Replace(basePath, "/", "_", -1)
	} else {
		name = args[1]
	}
	return &App{
		Injector: NewInjector(),
		Router:   NewRouter(basePath),
		Configs:  NewConfigs(),

		BasePath:  basePath,
		Name:      name,
		AppConfig: DefaultAppConfig,

		interceptors: make([]Interceptor, 0),
	}
}

func (a *App) Use(interceptors ...Interceptor) {
	for _, inter := range interceptors {
		a.interceptors = append(a.interceptors, inter)
		a.Map(inter)
	}
}

func (a *App) InjectAll() {
	for _, inter := range a.interceptors {
		a.Inject(inter)
	}
}

func (a *App) initApp() {
	// TODO: should test if logger has been mapped
	logger := a.Server.Logger
	a.Map(logger)

	a.Use(
		a.Configs,
		NewLogInterceptor(logger),
		NewPanicInterceptor(
			a.Server.Config.RecoverPanic,
			a.AppConfig.Mode == Debug,
		),
	)

	if a.Server.Config.EnableGzip {
		a.Use(NewCompressInterceptor(a.Server.Config.StaticExtensionsToGzip))
	}

	a.Use(
		&ReturnInterceptor{},
		&Static{
			RootPath: a.AppConfig.StaticDir,
			IndexFiles: []string{
				"index.html",
				"index.htm",
			},
		},
		&InitInterceptor{},
		&BeforeInterceptor{},
		&AfterInterceptor{},
		&RequestInterceptor{},
		&ResponseInterceptor{},
		&AppInterceptor{},
	)

	if a.AppConfig.FormMapToStruct {
		a.Use(&BindInterceptor{})
	}

	if a.AppConfig.StaticFileVersion {
		a.Use(NewStaticVersions(
			logger,
			a.AppConfig.StaticDir,
			a.basePath))
	} else {
		// even if don't use static file version, is still
		a.FuncMaps["StaticUrl"] = func(url string) string {
			return path.Join(a.basePath, url)
		}
	}

	render := NewRender(
		a.AppConfig.TemplateDir,
		a.AppConfig.ReloadTemplates,
		a.AppConfig.CacheTemplates,
	)
	a.Render = render
	a.Use(render)

	if a.AppConfig.CheckXsrf {
		a.Use(NewXsrfInterceptor())
	}

	if a.AppConfig.SessionOn {
		a.Use(NewSessionInterceptor(a))
		a.Map(a.SessionManager)
	}

	a.InjectAll()
}

func (a *App) SetStaticDir(dir string) {
	a.AppConfig.StaticDir = dir
}

func (a *App) SetTemplateDir(path string) {
	a.AppConfig.TemplateDir = path
}

func (a *App) routeHandler(req *http.Request, w http.ResponseWriter) {
	//ignore errors from ParseForm because it's usually harmless.
	ct := req.Header.Get("Content-Type")
	if strings.Contains(ct, "multipart/form-data") {
		req.ParseMultipartForm(a.AppConfig.MaxUploadSize)
	} else {
		req.ParseForm()
	}

	//set some default headers
	w.Header().Set("Server", "xweb")
	tm := time.Now().UTC()
	w.Header().Set("Date", webTime(tm))

	//Set the default content-type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	ctx := NewContext(
		a.Router,
		a.interceptors,
		req,
		NewResponseWriter(w),
	)

	ctx.Invoke()

	// flush the buffer
	ctx.Resp().Flush()
}

type AppInterface interface {
	SetApp(*App)
}

type AppInterceptor struct {
	app *App
}

func (inter *AppInterceptor) Intercept(ctx *Context) {
	action := ctx.Action()
	if action != nil {
		if apper, ok := action.(AppInterface); ok {
			apper.SetApp(inter.app)
		}
	}
	ctx.Invoke()
}
