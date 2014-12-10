package xweb

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/go-xweb/httpsession"
)

type App struct {
	*Injector
	*Router

	BasePath string
	Name     string //[SWH|+]

	Server    *Server
	AppConfig *AppConfig
	Config    map[string]interface{}

	FuncMaps template.FuncMap
	VarMaps  T

	// TODO: refactoring this
	SessionManager *httpsession.Manager //Session manager

	interceptors []Interceptor
}

const (
	Debug = iota + 1
	Product
)

type AppConfig struct {
	Mode              int
	StaticDir         string
	TemplateDir       string
	SessionOn         bool
	MaxUploadSize     int64
	CookieSecret      string
	StaticFileVersion bool
	CacheTemplates    bool
	ReloadTemplates   bool
	CheckXsrf         bool
	SessionTimeout    time.Duration
	FormMapToStruct   bool //[SWH|+]
	EnableHttpCache   bool //[SWH|+]
}

func NewApp(args ...string) *App {
	path := args[0]
	name := ""
	if len(args) == 1 {
		name = strings.Replace(path, "/", "_", -1)
	} else {
		name = args[1]
	}
	return &App{
		Injector: NewInjector(),
		Router:   NewRouter(),
		BasePath: path,
		Name:     name, //[SWH|+]
		AppConfig: &AppConfig{
			Mode:              Product,
			StaticDir:         "static",
			TemplateDir:       "templates",
			SessionOn:         true,
			SessionTimeout:    3600,
			MaxUploadSize:     10 * 1024 * 1024,
			StaticFileVersion: true,
			CacheTemplates:    true,
			ReloadTemplates:   true,
			CheckXsrf:         true,
			FormMapToStruct:   true,
		},
		Config: map[string]interface{}{},

		FuncMaps:     defaultFuncs,
		VarMaps:      T{},
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

	a.Use(NewRenderInterceptor(
		a.AppConfig.TemplateDir,
		a.AppConfig.ReloadTemplates,
		a.AppConfig.CacheTemplates,
		a,
	))

	if a.AppConfig.CheckXsrf {
		a.Use(NewXsrfInterceptor(a))
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

func (app *App) AddTmplVar(name string, varOrFun interface{}) {
	if reflect.TypeOf(varOrFun).Kind() == reflect.Func {
		app.FuncMaps[name] = varOrFun
	} else {
		app.VarMaps[name] = varOrFun
	}
}

func (app *App) AddTmplVars(t *T) {
	for name, value := range *t {
		app.AddTmplVar(name, value)
	}
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

	var ac = ActionContext{}
	ia := NewInvocation(
		a.Injector,
		a.interceptors,
		req,
		NewResponseWriter(w),
		&ac,
	)

	ac.newAction = func() {
		reqPath := removeStick(req.URL.Path)
		allowMethod := Ternary(req.Method == "HEAD", "GET", req.Method).(string)

		route, _, isFind := a.findRoute(reqPath, allowMethod)
		if isFind {
			ac.route = &route
			ac.action = a.newAction(ia, route).Interface()
		}
	}

	ac.Execute = func() interface{} {
		reqPath := removeStick(req.URL.Path)
		allowMethod := Ternary(req.Method == "HEAD", "GET", req.Method).(string)

		route, args, isFind := a.findRoute(reqPath, allowMethod)
		if !isFind {
			return nil
		}

		var vc reflect.Value
		if ia.action.action == nil {
			vc = a.newAction(ia, route)
			ia.action.action = vc.Interface()
		} else {
			vc = reflect.ValueOf(ia.action.action)
		}

		function := vc.MethodByName(route.HandlerMethod)
		ret := function.Call(args)

		if len(ret) > 0 {
			return ret[0].Interface()
		}
		return nil
	}

	ia.Invoke()

	// flush the buffer
	ia.Resp().Flush()
}

func (a *App) newAction(ia *Invocation, route Route) reflect.Value {
	vc := reflect.New(route.HandlerElement)

	if route.hasAction {
		c := &Action{
			C: vc,
		}

		vc.Elem().FieldByName("Action").Set(reflect.ValueOf(c))
	}

	return vc
}

func (a *App) error(w http.ResponseWriter, status int, content string) error {
	w.WriteHeader(status)
	if errorTmpl == "" {
		errTmplFile := a.AppConfig.TemplateDir + "/_error.html"
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
		status, statusText[status], content, Version)
	_, err := w.Write([]byte(res))
	return err
}

type AppInterface interface {
	SetApp(*App)
}

type AppInterceptor struct {
	app *App
}

func (inter *AppInterceptor) Intercept(ia *Invocation) {
	action := ia.ActionContext().Action()
	if action != nil {
		if apper, ok := action.(AppInterface); ok {
			apper.SetApp(inter.app)
		}
	}
	ia.Invoke()
}
