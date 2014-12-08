package xweb

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"reflect"
	"runtime"
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
	filters         []Filter
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
	RootTemplate    *template.Template
	ErrorTemplate   *template.Template
	StaticVerMgr    *StaticVerMgr
	TemplateMgr     *TemplateMgr
	interceptors    []Interceptor
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
		filters:         make([]Filter, 0),
		StaticVerMgr:    new(StaticVerMgr),
		TemplateMgr:     new(TemplateMgr),
		interceptors:    make([]Interceptor, 0),
	}
}

func (a *App) Use(interceptors ...Interceptor) {
	a.interceptors = append(a.interceptors, interceptors...)
}

func (a *App) initApp() {
	if a.Logger == nil {
		a.Logger = a.Server.Logger
	}

	a.Use(&LogInterceptor{})

	if a.Server.Config.EnableGzip {
		a.Use(&GZipInterceptor{})
	}

	a.Use(&ReturnInterceptor{})

	a.Use(&StaticInterceptor{
		RootPath: a.AppConfig.StaticDir,
		IndexFiles: []string{
			"index.html",
			"index.htm",
		},
	})

	a.Use(&InitInterceptor{},
		&BeforeInterceptor{},
		&AfterInterceptor{},
	)

	if a.AppConfig.FormMapToStruct {
		a.Use(&BindInterceptor{})
	}

	a.Use(&InjectInterceptor{})

	if a.AppConfig.StaticFileVersion {
		a.StaticVerMgr.Init(a, a.AppConfig.StaticDir)
		a.FuncMaps["StaticUrl"] = a.StaticUrl
	}

	if a.AppConfig.CacheTemplates {
		a.TemplateMgr.Init(a, a.AppConfig.TemplateDir, a.AppConfig.ReloadTemplates)
	}

	if a.AppConfig.CheckXsrf {
		a.Use(&XsrfInterceptor{})
		a.FuncMaps["XsrfName"] = XsrfName
	}

	a.VarMaps["XwebVer"] = Version

	if a.AppConfig.SessionOn {
		if a.Server.SessionManager != nil {
			a.SessionManager = a.Server.SessionManager
		} else {
			a.SessionManager = httpsession.Default()
			if a.AppConfig.SessionTimeout > time.Second {
				a.SessionManager.SetMaxAge(a.AppConfig.SessionTimeout)
			}
			a.SessionManager.Run()
		}
	}
}

func (a *App) SetStaticDir(dir string) {
	a.AppConfig.StaticDir = dir
}

func (a *App) SetTemplateDir(path string) {
	a.AppConfig.TemplateDir = path
}

func (a *App) getTemplatePath(name string) string {
	templateFile := path.Join(a.AppConfig.TemplateDir, name)
	if fileExists(templateFile) {
		return templateFile
	}
	return ""
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

func (app *App) filter(w http.ResponseWriter, req *http.Request) bool {
	for _, filter := range app.filters {
		if !filter.Do(w, req) {
			return false
		}
	}
	return true
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
	if !a.filter(w, req) {
		a.Info(req.Method, 302, req.URL.Path)
		return
	}

	var ac = ActionContext{}
	ia := NewInvocation(a, a.interceptors, req, NewResponseWriter(w), &ac)
	ia.SessionManager = a.SessionManager
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

		ret, err := a.SafelyCall(vc, route.HandlerMethod, args)
		if err != nil {
			return err
		}

		if len(ret) > 0 {
			return ret[0].Interface()
		} else {
			// if action return nil and not write to response, then return blank
			return ""
		}
	}

	ia.Invoke()

	// flush the buffer
	ia.Resp().Flush()
}

func (a *App) newAction(ia *Invocation, route Route) reflect.Value {
	vc := reflect.New(route.HandlerElement)
	c := &Action{
		Request:        ia.req,
		App:            a,
		ResponseWriter: ia.resp,
		T:              T{},
		f:              T{},
		Option: &ActionOption{
			AutoMapForm: a.AppConfig.FormMapToStruct,
			CheckXsrf:   a.AppConfig.CheckXsrf,
		},
	}

	for k, v := range a.VarMaps {
		c.T[k] = v
	}

	fieldA := vc.Elem().FieldByName("Action")
	if fieldA.IsValid() {
		fieldA.Set(reflect.ValueOf(c))
	}

	fieldC := vc.Elem().FieldByName("C")
	if fieldC.IsValid() {
		fieldC.Set(reflect.ValueOf(vc))
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

func (a *App) StaticUrl(url string) string {
	var basePath string
	if a.AppConfig.StaticDir == RootApp().AppConfig.StaticDir {
		basePath = RootApp().BasePath
	} else {
		basePath = a.BasePath
	}
	if !a.AppConfig.StaticFileVersion {
		return path.Join(basePath, url)
	}
	ver := a.StaticVerMgr.GetVersion(url)
	if ver == "" {
		return path.Join(basePath, url)
	}
	return path.Join(basePath, url+"?v="+ver)
}

// safelyCall invokes `function` in recover block
func (a *App) SafelyCall(vc reflect.Value, method string, args []reflect.Value) (resp []reflect.Value, err error) {
	defer func() {
		if e := recover(); e != nil {
			if !a.Server.Config.RecoverPanic {
				// go back to panic
				panic(e)
			} else {
				resp = nil
				var content string
				content = fmt.Sprintf("Handler crashed with error: %v", e)
				for i := 1; ; i += 1 {
					_, file, line, ok := runtime.Caller(i)
					if !ok {
						break
					} else {
						content += "\n"
					}
					content += fmt.Sprintf("%v %v", file, line)
				}
				a.Error(content)
				err = errors.New(content)
				return
			}
		}
	}()
	function := vc.MethodByName(method)
	return function.Call(args), err
}

var (
	sc *Action = &Action{}
)

func (app *App) Redirect(w http.ResponseWriter, requestPath, url string, status ...int) error {
	err := redirect(w, url, status...)
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
