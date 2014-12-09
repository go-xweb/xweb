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
	"github.com/go-xweb/log"
)

type App struct {
	BasePath        string
	Name            string //[SWH|+]
	Routes          []Route
	RoutesEq        map[string]map[string]Route
	Server          *Server
	AppConfig       *AppConfig
	Config          map[string]interface{}
	Actions         map[string]interface{}
	ActionsPath     map[reflect.Type]string
	ActionsNamePath map[string]string
	FuncMaps        template.FuncMap
	Logger          *log.Logger
	VarMaps         T
	SessionManager  *httpsession.Manager //Session manager

	//StaticVerMgr *StaticVerMgr
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
		BasePath: path,
		Name:     name, //[SWH|+]
		RoutesEq: make(map[string]map[string]Route),
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
		Config:          map[string]interface{}{},
		Actions:         map[string]interface{}{},
		ActionsPath:     map[reflect.Type]string{},
		ActionsNamePath: map[string]string{},
		FuncMaps:        defaultFuncs,
		VarMaps:         T{},
		//StaticVerMgr:    new(StaticVerMgr),
		interceptors: make([]Interceptor, 0),
	}
}

func (a *App) Use(interceptors ...Interceptor) {
	a.interceptors = append(a.interceptors, interceptors...)
}

func (a *App) initApp() {
	if a.Logger == nil {
		a.Logger = a.Server.Logger
	}

	a.Use(
		&LogInterceptor{},
		NewPanicInterceptor(a.AppConfig.Mode == Debug),
	)

	if a.Server.Config.EnableGzip {
		a.Use(&GZipInterceptor{})
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
		&InjectInterceptor{},
	)

	if a.AppConfig.FormMapToStruct {
		a.Use(&BindInterceptor{})
	}

	if a.AppConfig.StaticFileVersion {
		a.Use(NewStaticVerInterceptor(a, a.AppConfig.StaticDir))
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
	}
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

func (app *App) Debug(params ...interface{}) {
	args := append([]interface{}{"[" + app.Name + "]"}, params...)
	app.Logger.Debug(args...)
}

func (app *App) Info(params ...interface{}) {
	args := append([]interface{}{"[" + app.Name + "]"}, params...)
	app.Logger.Info(args...)
}

func (app *App) Warn(params ...interface{}) {
	args := append([]interface{}{"[" + app.Name + "]"}, params...)
	app.Logger.Warn(args...)
}

func (app *App) Error(params ...interface{}) {
	args := append([]interface{}{"[" + app.Name + "]"}, params...)
	app.Logger.Error(args...)
}

func (app *App) Fatal(params ...interface{}) {
	args := append([]interface{}{"[" + app.Name + "]"}, params...)
	app.Logger.Fatal(args...)
}

func (app *App) Panic(params ...interface{}) {
	args := append([]interface{}{"[" + app.Name + "]"}, params...)
	app.Logger.Panic(args...)
}

func (app *App) Debugf(format string, params ...interface{}) {
	app.Logger.Debugf("["+app.Name+"] "+format, params...)
}

func (app *App) Infof(format string, params ...interface{}) {
	app.Logger.Infof("["+app.Name+"] "+format, params...)
}

func (app *App) Warnf(format string, params ...interface{}) {
	app.Logger.Warnf("["+app.Name+"] "+format, params...)
}

func (app *App) Errorf(format string, params ...interface{}) {
	app.Logger.Errorf("["+app.Name+"] "+format, params...)
}

func (app *App) Fatalf(format string, params ...interface{}) {
	app.Logger.Fatalf("["+app.Name+"] "+format, params...)
}

func (app *App) Panicf(format string, params ...interface{}) {
	app.Logger.Panicf("["+app.Name+"] "+format, params...)
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
	ia := NewInvocation(a, a.interceptors, req, NewResponseWriter(w), &ac)

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
			Option: &ActionOption{
				AutoMapForm: a.AppConfig.FormMapToStruct,
				CheckXsrf:   a.AppConfig.CheckXsrf,
			},
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

var (
	sc *Action = &Action{}
)

func (app *App) Redirect(w http.ResponseWriter, requestPath, url string, status ...int) error {
	s := 302
	if len(status) > 0 {
		s = status[0]
	}
	w.Header().Set("Location", url)
	w.WriteHeader(s)
	_, err := w.Write([]byte("Redirecting to: " + url))
	if err != nil {
		app.Errorf("redirect error: %s", err)
		return err
	}
	return nil
}

func (app *App) Action(name string) interface{} {
	if v, ok := app.Actions[name]; ok {
		return v
	}
	return nil
}

/*
example:
{
	"AdminAction":{
		"Index":["GET","POST"],
		"Add":	["GET","POST"],
		"Edit":	["GET","POST"]
	}
}
*/
func (app *App) Nodes() (r map[string]map[string][]string) {
	r = make(map[string]map[string][]string)
	for _, val := range app.Routes {
		name := val.HandlerElement.Name()
		if _, ok := r[name]; !ok {
			r[name] = make(map[string][]string)
		}
		if _, ok := r[name][val.HandlerMethod]; !ok {
			r[name][val.HandlerMethod] = make([]string, 0)
		}
		for k, _ := range val.HttpMethods {
			r[name][val.HandlerMethod] = append(r[name][val.HandlerMethod], k) //FUNC1:[POST,GET]
		}
	}
	for _, vals := range app.RoutesEq {
		for k, v := range vals {
			name := v.HandlerElement.Name()
			if _, ok := r[name]; !ok {
				r[name] = make(map[string][]string)
			}
			if _, ok := r[name][v.HandlerMethod]; !ok {
				r[name][v.HandlerMethod] = make([]string, 0)
			}
			r[name][v.HandlerMethod] = append(r[name][v.HandlerMethod], k) //FUNC1:[POST,GET]
		}
	}
	return
}
