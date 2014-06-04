package xweb

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-xweb/httpsession"
	"github.com/go-xweb/log"
)

const (
	XSRF_TAG string = "_xsrf"
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
	Actions         map[reflect.Type]string
	FuncMaps        template.FuncMap
	Logger          *log.Logger
	VarMaps         T
	SessionManager  *httpsession.Manager //Session manager
	RootTemplate    *template.Template
	ErrorTemplate   *template.Template
	StaticVerMgr    *StaticVerMgr
	TemplateMgr     *TemplateMgr
	ContentEncoding string
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
	CheckXrsf         bool
	SessionTimeout    time.Duration
	FormMapToStruct   bool //[SWH|+]
	EnableHttpCache   bool //[SWH|+]

	//[SWH|+] On/Off Log
	EnableTraceLog    bool
	EnableDebugLog    bool
	EnableInfoLog     bool
	EnableWarnLog     bool
	EnableErrorLog    bool
	EnableCriticalLog bool
}

type Route struct {
	Path           string          //path string
	CompiledRegexp *regexp.Regexp  //path regexp
	HttpMethods    map[string]bool //GET POST HEAD DELETE etc.
	HandlerMethod  string          //struct method name
	HandlerElement reflect.Type    //handler element
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
			CheckXrsf:         true,
			FormMapToStruct:   true,

			EnableTraceLog:    true,
			EnableDebugLog:    true,
			EnableInfoLog:     true,
			EnableWarnLog:     true,
			EnableErrorLog:    true,
			EnableCriticalLog: true,
		},
		Config:       map[string]interface{}{},
		Actions:      map[reflect.Type]string{},
		FuncMaps:     defaultFuncs,
		VarMaps:      T{},
		filters:      make([]Filter, 0),
		StaticVerMgr: new(StaticVerMgr),
		TemplateMgr:  new(TemplateMgr),
	}
}

func (a *App) initApp() {
	if a.AppConfig.StaticFileVersion {
		a.StaticVerMgr.Init(a, a.AppConfig.StaticDir)
	}
	if a.AppConfig.CacheTemplates {
		a.TemplateMgr.Init(a, a.AppConfig.TemplateDir, a.AppConfig.ReloadTemplates)
	}
	a.FuncMaps["StaticUrl"] = a.StaticUrl
	a.FuncMaps["XsrfName"] = XsrfName
	a.VarMaps["XwebVer"] = Version

	if a.AppConfig.SessionOn {
		a.SessionManager = httpsession.Default()
		a.SessionManager.SetMaxAge(a.AppConfig.SessionTimeout)
		a.SessionManager.Run()
	}

	if a.Logger == nil {
		a.Logger = a.Server.Logger
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

func (app *App) AddAction(cs ...interface{}) {
	for _, c := range cs {
		app.AddRouter(app.BasePath, c)
	}
}

func (app *App) AutoAction(cs ...interface{}) {
	for _, c := range cs {
		t := reflect.Indirect(reflect.ValueOf(c)).Type()
		name := t.Name()
		if strings.HasSuffix(name, "Action") {
			path := strings.ToLower(name[:len(name)-6])
			app.AddRouter(JoinPath(app.BasePath, path), c)
		}
	}
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

func (app *App) AddFilter(filter Filter) {
	app.filters = append(app.filters, filter)
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

func (a *App) addRoute(r string, methods map[string]bool, t reflect.Type, handler string) {
	cr, err := regexp.Compile(r)
	if err != nil {
		a.Errorf("Error in route regex %q: %s", r, err)
		return
	}
	a.Routes = append(a.Routes, Route{Path: r, CompiledRegexp: cr, HttpMethods: methods, HandlerMethod: handler, HandlerElement: t})
}

func (a *App) addEqRoute(r string, methods map[string]bool, t reflect.Type, handler string) {
	if _, ok := a.RoutesEq[r]; !ok {
		a.RoutesEq[r] = make(map[string]Route)
	}
	for v, _ := range methods {
		a.RoutesEq[r][v] = Route{HandlerMethod: handler, HandlerElement: t}
	}
}

var (
	mapperType = reflect.TypeOf(Mapper{})
)

func (app *App) AddRouter(url string, c interface{}) {
	t := reflect.TypeOf(c).Elem()
	app.Actions[t] = url
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Type != mapperType {
			continue
		}
		name := t.Field(i).Name
		a := strings.Title(name)
		v := reflect.ValueOf(c).MethodByName(a)
		if !v.IsValid() {
			continue
		}

		tag := t.Field(i).Tag
		tagStr := tag.Get("xweb")
		methods := map[string]bool{"GET": true, "POST": true}
		var p string
		var isEq bool
		if tagStr != "" {
			tags := strings.Split(tagStr, " ")
			path := tagStr
			length := len(tags)
			if length >= 2 {
				for _, method := range strings.Split(tags[0], "|") {
					methods[strings.ToUpper(method)] = true
				}
				path = tags[1]
				if regexp.QuoteMeta(path) == path {
					isEq = true
				}
			} else if length == 1 {
				if tags[0][0] == '/' {
					path = tags[0]
					if regexp.QuoteMeta(path) == path {
						isEq = true
					}
				} else {
					for _, method := range strings.Split(tags[0], "|") {
						methods[strings.ToUpper(method)] = true
					}
					path = "/" + name
					isEq = true
				}
			} else {
				path = "/" + name
				isEq = true
			}
			p = strings.TrimRight(url, "/") + path
		} else {
			p = strings.TrimRight(url, "/") + "/" + name
			isEq = true
		}
		if isEq {
			app.addEqRoute(removeStick(p), methods, t, a)
		} else {
			app.addRoute(removeStick(p), methods, t, a)
		}
	}
}

// the main route handler in web.go
func (a *App) routeHandler(req *http.Request, w http.ResponseWriter) {
	requestPath := req.URL.Path
	var statusCode = 0
	defer func() {
		if statusCode == 0 {
			statusCode = 200
		}
		if statusCode >= 200 && statusCode < 400 {
			a.Info(req.Method, statusCode, requestPath)
		} else {
			a.Error(req.Method, statusCode, requestPath)
		}
	}()

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

	// static files, needed op
	if req.Method == "GET" || req.Method == "HEAD" {
		success := a.TryServingFile(requestPath, req, w)
		if success {
			statusCode = 200
			return
		}
		if requestPath == "/favicon.ico" {
			statusCode = 404
			a.error(w, 404, "Page not found")
			return
		}
	}

	//Set the default content-type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if !a.filter(w, req) {
		statusCode = 302
		return
	}
	requestPath = req.URL.Path //[SWH|+]support filter change req.URL.Path

	reqPath := removeStick(requestPath)
	allowMethod := Ternary(req.Method == "HEAD", "GET", req.Method).(string)
	isFind := false
	if routes, ok := a.RoutesEq[reqPath]; ok {
		if route, ok := routes[allowMethod]; ok {
			var isBreak bool = false
			var args []reflect.Value
			isBreak, statusCode = a.run(req, w, route, args)
			if isBreak {
				return
			}
			isFind = true
		}
	}
	if !isFind {
		for _, route := range a.Routes {
			cr := route.CompiledRegexp

			//if the methods don't match, skip this handler (except HEAD can be used in place of GET)
			if _, ok := route.HttpMethods[allowMethod]; !ok {
				continue
			}

			if !cr.MatchString(reqPath) {
				continue
			}

			match := cr.FindStringSubmatch(reqPath)

			if len(match[0]) != len(reqPath) {
				continue
			}

			var args []reflect.Value
			for _, arg := range match[1:] {
				args = append(args, reflect.ValueOf(arg))
			}
			var isBreak bool = false
			isBreak, statusCode = a.run(req, w, route, args)
			if isBreak {
				return
			}
		}
	}
	// try serving index.html or index.htm
	if req.Method == "GET" || req.Method == "HEAD" {
		if a.TryServingFile(path.Join(requestPath, "index.html"), req, w) {
			statusCode = 200
			return
		} else if a.TryServingFile(path.Join(requestPath, "index.htm"), req, w) {
			statusCode = 200
			return
		}
	}

	a.error(w, 404, "Page not found")
	statusCode = 404
}

func (a *App) run(req *http.Request, w http.ResponseWriter, route Route, args []reflect.Value) (isBreak bool, statusCode int) {

	vc := reflect.New(route.HandlerElement)
	c := &Action{
		Request:        req,
		App:            a,
		ResponseWriter: w,
		T:              T{},
		f:              T{},
		Option: &ActionOption{
			AutoMapForm: a.AppConfig.FormMapToStruct,
			CheckXrsf:   a.AppConfig.CheckXrsf,
		},
	}

	for k, v := range a.VarMaps {
		c.T[k] = v
	}

	fieldA := vc.Elem().FieldByName("Action")
	//fieldA := fieldByName(vc.Elem(), "Action")
	if fieldA.IsValid() {
		fieldA.Set(reflect.ValueOf(c))
	}

	fieldC := vc.Elem().FieldByName("C")
	//fieldC := fieldByName(vc.Elem(), "C")
	if fieldC.IsValid() {
		fieldC.Set(reflect.ValueOf(vc))
		//fieldC.Set(vc)
	}

	initM := vc.MethodByName("Init")
	if initM.IsValid() {
		params := []reflect.Value{}
		initM.Call(params)
	}

	if c.Option.AutoMapForm {
		a.StructMap(vc.Elem(), req)
	}

	if c.Option.CheckXrsf && req.Method == "POST" {
		res, err := req.Cookie(XSRF_TAG)
		formVals := req.Form[XSRF_TAG]
		var formVal string
		if len(formVals) > 0 {
			formVal = formVals[0]
		}
		if err != nil || res.Value == "" || res.Value != formVal {
			a.error(w, 500, "xrsf token error.")
			a.Error("xrsf token error.")
			statusCode = 500
			isBreak = true
			return
		}
	}

	//[SWH|+]------------------------------------------Before-Hook
	structName := reflect.ValueOf(route.HandlerElement.Name())
	actionName := reflect.ValueOf(route.HandlerMethod)
	initM = vc.MethodByName("Before")
	if initM.IsValid() {
		structAction := []reflect.Value{structName, actionName}
		if ok := initM.Call(structAction); !ok[0].Bool() {
			isBreak = true
			return
		}
	}

	ret, err := a.SafelyCall(vc, route.HandlerMethod, args)
	if err != nil {
		a.error(w, 500, fmt.Sprintf("handler error: %v", err))
		//there was an error or panic while calling the handler
		if a.AppConfig.Mode == Debug {
			a.error(w, 500, err.Error())
		} else if a.AppConfig.Mode == Product {
			a.error(w, 500, "Server Error")
		}
		statusCode = 500
		isBreak = true
		return
	}
	statusCode = fieldA.Interface().(*Action).StatusCode

	//[SWH|+]------------------------------------------After-Hook
	initM = vc.MethodByName("After")
	if initM.IsValid() {
		structAction := []reflect.Value{structName, actionName}
		for _, v := range ret {
			structAction = append(structAction, v)
		}
		if len(structAction) != initM.Type().NumIn() {
			a.Error("Error : %v.After(): The number of params is not adapted.", structName)
			isBreak = true
			return
		}
		if ok := initM.Call(structAction); !ok[0].Bool() {
			isBreak = true
			return
		}
	}

	if len(ret) == 0 {
		isBreak = true
		return
	}

	sval := ret[0]

	var content []byte
	if sval.Interface() == nil || sval.Kind() == reflect.Bool {
		isBreak = true
		return
	} else if sval.Kind() == reflect.String {
		content = []byte(sval.String())
	} else if sval.Kind() == reflect.Slice && sval.Type().Elem().Kind() == reflect.Uint8 {
		content = sval.Interface().([]byte)
	} else if err, ok := sval.Interface().(error); ok {
		if err != nil {
			a.Error("Error:", err)
			a.error(w, 500, "Server Error")
			statusCode = 500
		}
		isBreak = true
		return
	} else {
		a.Warn("unkonw returned result type %v, ignored %v", sval,
			sval.Interface().(error))

		isBreak = true
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	_, err = w.Write(content)
	if err != nil {
		a.Error("Error during write: %v", err)
		statusCode = 500
		isBreak = true
		return
	}
	return
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

// Init content-length header.
func (a *App) InitHeadContent(w http.ResponseWriter, contentLength int64) {
	if a.ContentEncoding == "gzip" {
		w.Header().Set("Content-Encoding", "gzip")
	} else if a.ContentEncoding == "deflate" {
		w.Header().Set("Content-Encoding", "deflate")
	} else {
		w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	}
}

// tryServingFile attempts to serve a static file, and returns
// whether or not the operation is successful.
func (a *App) TryServingFile(name string, req *http.Request, w http.ResponseWriter) bool {
	newPath := name
	if strings.HasPrefix(name, a.BasePath) {
		newPath = name[len(a.BasePath):]
	}
	staticFile := filepath.Join(a.AppConfig.StaticDir, newPath)
	finfo, err := os.Stat(staticFile)
	if err != nil {
		return false
	}
	if !finfo.IsDir() {
		isStaticFileToCompress := false
		if a.Server.Config.EnableGzip && a.Server.Config.StaticExtensionsToGzip != nil && len(a.Server.Config.StaticExtensionsToGzip) > 0 {
			for _, statExtension := range a.Server.Config.StaticExtensionsToGzip {
				if strings.HasSuffix(strings.ToLower(staticFile), strings.ToLower(statExtension)) {
					isStaticFileToCompress = true
					break
				}
			}
		}
		if isStaticFileToCompress {
			a.ContentEncoding = GetAcceptEncodingZip(req)
			memzipfile, err := OpenMemZipFile(staticFile, a.ContentEncoding)
			if err != nil {
				return false
			}
			a.InitHeadContent(w, finfo.Size())
			http.ServeContent(w, req, staticFile, finfo.ModTime(), memzipfile)
		} else {
			http.ServeFile(w, req, staticFile)
		}
		return true
	}
	return false
}

var (
	sc *Action = &Action{}
)

// StructMap function mapping params to controller's properties
func (a *App) StructMap(vc reflect.Value, r *http.Request) error {
	return a.namedStructMap(vc, r, "")
}

func (a *App) namedStructMap(vc reflect.Value, r *http.Request, topName string) error {
	for k, t := range r.Form {
		if k == XSRF_TAG || k == "" {
			continue
		}

		if topName != "" {
			if !strings.HasPrefix(k, topName) {
				continue
			}
			k = k[len(topName)+1:]
		}

		v := t[0]
		names := strings.Split(k, ".")
		var value reflect.Value = vc
		for i, name := range names {
			name = strings.Title(name)
			if i != len(names)-1 {
				if value.Kind() != reflect.Struct {
					a.Warn("arg error, value kind is %v", value.Kind())
					break
				}

				//fmt.Println(name)
				value = value.FieldByName(name)
				if !value.IsValid() {
					a.Warn("(%v value is not valid %v)", name, value)
					break
				}
				if !value.CanSet() {
					a.Warn("can not set %v -> %v", name, value.Interface())
					break
				}

				if value.Kind() == reflect.Ptr {
					if value.IsNil() {
						value.Set(reflect.New(value.Type().Elem()))
					}
					value = value.Elem()
				}
			} else {
				tv := value.FieldByName(name)
				if !tv.IsValid() {
					break
				}
				if !tv.CanSet() {
					a.Warn("can not set %v to %v", k, tv)
					break
				}

				if tv.Kind() == reflect.Ptr {
					tv.Set(reflect.New(tv.Type().Elem()))
					tv = tv.Elem()
				}

				var l interface{}
				switch k := tv.Kind(); k {
				case reflect.String:
					l = v
					tv.Set(reflect.ValueOf(l))
				case reflect.Bool:
					l = (v != "false" && v != "0")
					tv.Set(reflect.ValueOf(l))
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
					x, err := strconv.Atoi(v)
					if err != nil {
						a.Warn("arg %v as int: %v", v, err)
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Int64:
					x, err := strconv.ParseInt(v, 10, 64)
					if err != nil {
						a.Warn("arg %v as int64: %v", v, err)
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Float32, reflect.Float64:
					x, err := strconv.ParseFloat(v, 64)
					if err != nil {
						a.Warn("arg %v as float64: %v", v, err)
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					x, err := strconv.ParseUint(v, 10, 64)
					if err != nil {
						a.Warn("arg %v as uint: %v", v, err)
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Struct:
					if tvf, ok := tv.Interface().(FromConversion); ok {
						err := tvf.FromString(v)
						if err != nil {
							a.Warn("struct %v invoke FromString faild", tvf)
						}
					} else if tv.Type().String() == "time.Time" {
						x, err := time.Parse("2006-01-02 15:04:05.000 -0700", v)
						if err != nil {
							x, err = time.Parse("2006-01-02 15:04:05", v)
							if err != nil {
								x, err = time.Parse("2006-01-02", v)
								if err != nil {
									a.Warn("unsupported time format %v, %v", v, err)
									break
								}
							}
						}
						l = x
						tv.Set(reflect.ValueOf(l))
					} else {
						a.Warn("can not set an struct which is not implement Fromconversion interface")
					}
				case reflect.Ptr:
					a.Warn("can not set an ptr of ptr")
				case reflect.Slice, reflect.Array:
					tt := tv.Type().Elem()
					tk := tt.Kind()
					if tk == reflect.String {
						tv.Set(reflect.ValueOf(t))
						break
					}

					if tv.IsNil() {
						tv.Set(reflect.MakeSlice(tv.Type(), len(t), len(t)))
					}

					for i, s := range t {
						var err error
						switch tk {
						case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int8, reflect.Int64:
							var v int64
							v, err = strconv.ParseInt(s, 10, tt.Bits())
							if err == nil {
								tv.Index(i).SetInt(v)
							}
						case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
							var v uint64
							v, err = strconv.ParseUint(s, 10, tt.Bits())
							if err == nil {
								tv.Index(i).SetUint(v)
							}
						case reflect.Float32, reflect.Float64:
							var v float64
							v, err = strconv.ParseFloat(s, tt.Bits())
							if err == nil {
								tv.Index(i).SetFloat(v)
							}
						case reflect.Bool:
							var v bool
							v, err = strconv.ParseBool(s)
							if err == nil {
								tv.Index(i).SetBool(v)
							}
						case reflect.Complex64, reflect.Complex128:
							// TODO:
							err = fmt.Errorf("unsupported slice element type %v", tk.String())
						default:
							err = fmt.Errorf("unsupported slice element type %v", tk.String())
						}
						if err != nil {
							a.Warn("slice error: %v, %v", name, err)
							break
						}
					}
				default:
					break
				}
			}
		}
	}
	return nil
}

func (app *App) Redirect(w http.ResponseWriter, requestPath, url string, status ...int) error {
	err := redirect(w, url, status...)
	if err != nil {
		app.Error("redirect error: %s", err)
		return err
	}
	return nil
}

func (app *App) Nodes() (r map[string]map[string]string) {
	r = make(map[string]map[string]string)
	for _, val := range app.Routes {
		name := val.HandlerElement.Name()
		if _, ok := r[name]; !ok {
			r[name] = make(map[string]string)
		}
		for k, _ := range val.HttpMethods {
			r[name][k] = val.HandlerMethod
		}
	}
	for _, vals := range app.RoutesEq {
		for k, v := range vals {
			name := v.HandlerElement.Name()
			if _, ok := r[name]; !ok {
				r[name] = make(map[string]string)
			}
			r[name][k] = v.HandlerMethod
		}
	}
	return
}
