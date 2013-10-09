package xweb

import (
	"bytes"
	"fmt"
	"github.com/astaxie/beego/session"
	"html/template"
	"net/http"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	XSRF_TAG string = "_xsrf"
)

type App struct {
	BasePath       string
	routes         []route
	filters        []Filter
	Server         *Server
	AppConfig      *AppConfig
	Config         map[string]interface{}
	Actions        map[reflect.Type]string
	FuncMaps       template.FuncMap
	VarMaps        T
	SessionManager *session.Manager //Session manager
	RootTemplate   *template.Template
	ErrorTemplate  *template.Template
	StaticVerMgr   *StaticVerMgr
	TemplateMgr    *TemplateMgr
}

type AppConfig struct {
	StaticDir         string
	TemplateDir       string
	SessionOn         bool
	MaxUploadSize     int64
	CookieSecret      string
	StaticFileVersion bool
	CacheTemplates    bool
	ReloadTemplates   bool
	CheckXrsf         bool
	SessionTimeout    int64
}

type route struct {
	r       string
	cr      *regexp.Regexp
	methods map[string]bool
	handler string
	ctype   reflect.Type
}

func NewApp(path string) *App {
	return &App{BasePath: path,
		AppConfig: &AppConfig{
			StaticDir:         "static",
			TemplateDir:       "templates",
			SessionOn:         true,
			SessionTimeout:    3600,
			MaxUploadSize:     10 * 1024 * 1024,
			StaticFileVersion: true,
			CacheTemplates:    true,
			ReloadTemplates:   true,
			CheckXrsf:         true,
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
		a.StaticVerMgr.Init(a.AppConfig.StaticDir)
	}
	if a.AppConfig.CacheTemplates {
		a.TemplateMgr.Init(a.AppConfig.TemplateDir, a.AppConfig.ReloadTemplates)
	}
	a.FuncMaps["StaticUrl"] = a.StaticUrl
	a.FuncMaps["XsrfName"] = XsrfName

	if a.AppConfig.SessionOn {
		identify := fmt.Sprintf("xweb_%v_%v_%v", a.Server.Config.Addr,
			a.Server.Config.Port, strings.Replace(a.BasePath, "/", "_", -1))
		//fmt.Println(identify)
		a.SessionManager, _ = session.NewManager("memory", identify, a.AppConfig.SessionTimeout, "")
		go a.SessionManager.GC()
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

func (app *App) AddFilter(filter Filter) {
	app.filters = append(app.filters, filter)
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
		a.Server.Logger.Printf("Error in route regex %q\n", r)
		return
	}

	a.routes = append(a.routes, route{r, cr, methods, handler, t})
}

func (app *App) AddRouter(url string, c interface{}) {
	t := reflect.TypeOf(c).Elem()
	app.Actions[t] = url
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Type != reflect.TypeOf(Mapper{}) {
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
		if tagStr != "" {
			tags := strings.Split(tagStr, " ")
			path := tagStr
			if len(tags) >= 2 {
				for _, method := range strings.Split(tags[0], "|") {
					methods[strings.ToUpper(method)] = true
				}
				path = tags[1]
			} else if len(tags) == 1 {
				if strings.HasPrefix(tags[0], "/") {
					path = tags[0]
				} else {
					for _, method := range strings.Split(tags[0], "|") {
						methods[strings.ToUpper(method)] = true
					}
					path = "/" + name
				}
			} else {
				path = "/" + name
			}

			app.addRoute(strings.TrimRight(url, "/")+path, methods, t, a)
		} else {
			app.addRoute(strings.TrimRight(url, "/")+"/"+name, methods, t, a)
		}
	}
}

// the main route handler in web.go
func (a *App) routeHandler(req *http.Request, w http.ResponseWriter) {
	requestPath := req.URL.Path

	//log the request
	var logEntry bytes.Buffer
	fmt.Fprintf(&logEntry, "\033[32;1m%s %s\033[0m", req.Method, requestPath)

	//ignore errors from ParseForm because it's usually harmless.
	ct := req.Header.Get("Content-Type")
	if strings.Contains(ct, "multipart/form-data") {
		req.ParseMultipartForm(a.AppConfig.MaxUploadSize)
	} else {
		req.ParseForm()
	}

	a.Server.Logger.Print(logEntry.String())

	//set some default headers
	w.Header().Set("Server", "xweb")
	tm := time.Now().UTC()
	w.Header().Set("Date", webTime(tm))

	// static files, needed op
	if req.Method == "GET" || req.Method == "HEAD" {
		if a.tryServingFile(requestPath, req, w) {
			return
		}
	}

	//Set the default content-type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if !a.filter(w, req) {
		return
	}

	for i := 0; i < len(a.routes); i++ {
		route := a.routes[i]
		cr := route.cr
		//if the methods don't match, skip this handler (except HEAD can be used in place of GET)
		allowMethod := Ternary(req.Method == "HEAD", "GET", req.Method).(string)
		if _, ok := route.methods[allowMethod]; !ok {
			continue
		}

		if !cr.MatchString(requestPath) {
			continue
		}
		match := cr.FindStringSubmatch(requestPath)

		if len(match[0]) != len(requestPath) {
			continue
		}

		if a.AppConfig.CheckXrsf && req.Method == "POST" {
			res, err := req.Cookie(XSRF_TAG)
			formVals := req.Form[XSRF_TAG]
			var formVal string
			if len(formVals) > 0 {
				formVal = formVals[0]
			}
			if err != nil || res.Value == "" || res.Value != formVal {
				w.WriteHeader(500)
				w.Write([]byte("xrsf error."))
				return
			}
		}

		var args []reflect.Value
		for _, arg := range match[1:] {
			args = append(args, reflect.ValueOf(arg))
		}
		vc := reflect.New(route.ctype)
		c := Action{Request: req, App: a, ResponseWriter: w, T: T{}, f: T{}}
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

		a.StructMap(vc.Elem(), req)

		initM := vc.MethodByName("Init")
		if initM.IsValid() {
			params := []reflect.Value{}
			initM.Call(params)
		}

		ret, err := a.safelyCall(vc, route.handler, args)
		if err != nil {
			c.GetLogger().Println(err)
			//there was an error or panic while calling the handler
			c.Abort(500, "Server Error")
			return
		}
		if len(ret) == 0 {
			return
		}

		sval := ret[0]

		var content []byte
		if sval.Kind() == reflect.String {
			content = []byte(sval.String())
		} else if sval.Kind() == reflect.Slice && sval.Type().Elem().Kind() == reflect.Uint8 {
			content = sval.Interface().([]byte)
		} else if e, ok := sval.Interface().(error); ok && e != nil {
			c.GetLogger().Println(e)
			c.Abort(500, "Server Error")
			return
		}
		c.SetHeader("Content-Length", strconv.Itoa(len(content)))
		_, err = c.ResponseWriter.Write(content)
		if err != nil {
			a.Server.Logger.Println("Error during write: ", err)
		}
		return
	}

	// try serving index.html or index.htm
	if req.Method == "GET" || req.Method == "HEAD" {
		if a.tryServingFile(path.Join(requestPath, "index.html"), req, w) {
			return
		} else if a.tryServingFile(path.Join(requestPath, "index.htm"), req, w) {
			return
		}
	}
	w.WriteHeader(404)
	w.Write([]byte("Page not found"))
}

func (a *App) StaticUrl(url string) string {
	if !a.AppConfig.StaticFileVersion {
		return a.BasePath + url
	}
	ver := a.StaticVerMgr.GetVersion(url)
	if ver == "" {
		return a.BasePath + url
	}
	return a.BasePath + url + "?v=" + ver
}

// safelyCall invokes `function` in recover block
func (a *App) safelyCall(vc reflect.Value, method string, args []reflect.Value) (resp []reflect.Value, e interface{}) {
	defer func() {
		if err := recover(); err != nil {
			if !a.Server.Config.RecoverPanic {
				// go back to panic
				panic(err)
			} else {
				e = err
				resp = nil
				a.Server.Logger.Println("Handler crashed with error", err)
				for i := 1; ; i += 1 {
					_, file, line, ok := runtime.Caller(i)
					if !ok {
						break
					}
					a.Server.Logger.Println(file, line)
				}
			}
		}
	}()
	function := vc.MethodByName(method)
	return function.Call(args), nil
}

// tryServingFile attempts to serve a static file, and returns
// whether or not the operation is successful.
func (a *App) tryServingFile(name string, req *http.Request, w http.ResponseWriter) bool {
	newPath := name[len(a.BasePath):]
	staticFile := path.Join(a.AppConfig.StaticDir, newPath)
	if fileExists(staticFile) {
		http.ServeFile(w, req, staticFile)
		return true
	}
	//fmt.Println(name)
	return false
}

var (
	sc *Action = &Action{}
)

/*func FormMap(prefix string, result interface{}, values *map[string][]string) error {
	value := reflect.ValueOf(result)
	value.

	for k, t := range values {
		names := strings.Split(k, ".")
		var value reflect.Value = vc
		for i, name := range names {
			name = strings.Title(name)
			if i == 0 {
				if reflect.ValueOf(sc).Elem().FieldByName(name).IsValid() {
					a.Server.Logger.Printf("Controller's property should not be changed by mapper.")
					break
				}
			}
			if value.Kind() != reflect.Struct {
				a.Server.Logger.Printf("arg error, value kind is %v", value.Kind())
				break
			}

			if i != len(names)-1 {
				value = value.FieldByName(name)
				if !value.IsValid() {
					a.Server.Logger.Printf("(%v value is not valid %v)", name, value)
					break
				}
			} else {
				tv := value.FieldByName(name)
				if !tv.IsValid() {
					//a.Server.Logger.Printf("struct %v has no field named %v", value, name)
					break
				}
				if !tv.CanSet() {
					a.Server.Logger.Printf("can not set %v", k)
					break
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
						a.Server.Logger.Printf("arg " + v + " as int: " + err.Error())
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Int64:
					x, err := strconv.ParseInt(v, 10, 64)
					if err != nil {
						a.Server.Logger.Printf("arg " + v + " as int: " + err.Error())
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Float32, reflect.Float64:
					x, err := strconv.ParseFloat(v, 64)
					if err != nil {
						a.Server.Logger.Printf("arg " + v + " as float64: " + err.Error())
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					x, err := strconv.ParseUint(v, 10, 64)
					if err != nil {
						a.Server.Logger.Printf("arg " + v + " as int: " + err.Error())
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Struct:
					if tvf, ok := tv.Interface().(FromConversion); ok {
						err := tvf.FromString(v)
						if err != nil {
							a.Server.Logger.Printf("struct %v invoke FromString faild", tvf)
						}
					} else {
						a.Server.Logger.Printf("can not set an struct which is not implement Fromconversion interface")
					}
				case reflect.Ptr:
					a.Server.Logger.Printf("can not set an ptr")
				case reflect.Slice, reflect.Array:
					a.Server.Logger.Printf("slice or array %v", t)
					tv.Set(reflect.ValueOf(t))
				default:
					break
				}
			}
		}
}*/

// StructMap function mapping params to controller's properties
func (a *App) StructMap(vc reflect.Value, r *http.Request) error {
	for k, t := range r.Form {
		v := t[0]
		names := strings.Split(k, ".")
		var value reflect.Value = vc
		for i, name := range names {
			name = strings.Title(name)
			if i == 0 {
				if reflect.ValueOf(sc).Elem().FieldByName(name).IsValid() {
					a.Server.Logger.Printf("Controller's property should not be changed by mapper.")
					break
				}
			}
			if value.Kind() != reflect.Struct {
				a.Server.Logger.Printf("arg error, value kind is %v", value.Kind())
				break
			}

			if i != len(names)-1 {
				value = value.FieldByName(name)
				if !value.IsValid() {
					a.Server.Logger.Printf("(%v value is not valid %v)", name, value)
					break
				}
			} else {
				tv := value.FieldByName(name)
				if !tv.IsValid() {
					//a.Server.Logger.Printf("struct %v has no field named %v", value, name)
					break
				}
				if !tv.CanSet() {
					a.Server.Logger.Printf("can not set %v", k)
					break
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
						a.Server.Logger.Printf("arg " + v + " as int: " + err.Error())
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Int64:
					x, err := strconv.ParseInt(v, 10, 64)
					if err != nil {
						a.Server.Logger.Printf("arg " + v + " as int: " + err.Error())
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Float32, reflect.Float64:
					x, err := strconv.ParseFloat(v, 64)
					if err != nil {
						a.Server.Logger.Printf("arg " + v + " as float64: " + err.Error())
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					x, err := strconv.ParseUint(v, 10, 64)
					if err != nil {
						a.Server.Logger.Printf("arg " + v + " as int: " + err.Error())
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Struct:
					if tvf, ok := tv.Interface().(FromConversion); ok {
						err := tvf.FromString(v)
						if err != nil {
							a.Server.Logger.Printf("struct %v invoke FromString faild", tvf)
						}
					} else {
						a.Server.Logger.Printf("can not set an struct which is not implement Fromconversion interface")
					}
				case reflect.Ptr:
					a.Server.Logger.Printf("can not set an ptr")
				case reflect.Slice, reflect.Array:
					a.Server.Logger.Printf("slice or array %v", t)
					tv.Set(reflect.ValueOf(t))
				default:
					break
				}
			}
		}
	}
	return nil
}
