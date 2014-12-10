package xweb

import (
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/go-xweb/httpsession"
)

type App struct {
	*Injector
	*Router
	*Render

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
	path := args[0]
	name := ""
	if len(args) == 1 {
		name = strings.Replace(path, "/", "_", -1)
	} else {
		name = args[1]
	}
	return &App{
		Injector:  NewInjector(),
		Router:    NewRouter(),
		BasePath:  path,
		Name:      name, //[SWH|+]
		AppConfig: DefaultAppConfig,
		Config:    map[string]interface{}{},

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
		&StaticInterceptor{
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
		a.Use(NewStaticVerInterceptor(logger, a.AppConfig.StaticDir, a))
	} else {
		// even if don't use static file version, is still
		a.FuncMaps["StaticUrl"] = a.StaticUrlNoVer
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

func (app *App) SetConfig(name string, val interface{}) {
	app.Config[name] = val
}

func (app *App) GetConfig(name string) interface{} {
	return app.Config[name]
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
		a.Injector,
		a.interceptors,
		req,
		NewResponseWriter(w),
	)

	ctx.newAction = func() {
		reqPath := removeStick(req.URL.Path)
		allowMethod := Ternary(req.Method == "HEAD", "GET", req.Method).(string)

		route, _, isFind := a.findRoute(reqPath, allowMethod)
		if isFind {
			ctx.route = &route
			ctx.action = a.newAction(ctx, route).Interface()
		}
	}

	ctx.Execute = func() interface{} {
		reqPath := removeStick(req.URL.Path)
		allowMethod := Ternary(req.Method == "HEAD", "GET", req.Method).(string)

		route, args, isFind := a.findRoute(reqPath, allowMethod)
		if !isFind {
			return nil
		}

		var vc reflect.Value
		if ctx.action == nil {
			vc = a.newAction(ctx, route)
			ctx.action = vc.Interface()
		} else {
			vc = reflect.ValueOf(ctx.action)
		}

		function := vc.MethodByName(route.HandlerMethod)
		ret := function.Call(args)

		if len(ret) > 0 {
			return ret[0].Interface()
		}
		return nil
	}

	ctx.Invoke()

	// flush the buffer
	ctx.Resp().Flush()
}

func (a *App) newAction(ctx *Context, route Route) reflect.Value {
	vc := reflect.New(route.HandlerElement)

	if route.hasAction {
		c := &Action{
			C: vc,
		}

		vc.Elem().FieldByName("Action").Set(reflect.ValueOf(c))
	}

	return vc
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
