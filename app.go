package xweb

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/lunny/httpsession"
)

const (
	XSRF_TAG string = "_xsrf"
)

type App struct {
	BasePath        string
	Name            string //[SWH|+]
	routes          []route
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
	SessionTimeout    int64
	FormMapToStruct   bool //[SWH|+]
	EnableHttpCache   bool //[SWH|+]
}

type route struct {
	r       string
	cr      *regexp.Regexp
	methods map[string]bool
	handler string
	ctype   reflect.Type
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

	if a.AppConfig.SessionOn {
		a.SessionManager = httpsession.Default()
		a.SessionManager.Run()
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
		a.Logger.Printf("Error in route regex %q\n", r)
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

const (
	ForeBlack  = iota + 30 //30         40         黑色
	ForeRed                //31         41         紅色
	ForeGreen              //32         42         綠色
	ForeYellow             //33         43         黃色
	ForeBlue               //34         44         藍色
	ForePurple             //35         45         紫紅色
	ForeCyan               //36         46         青藍色
	ForeWhite              //37         47         白色
)

// the main route handler in web.go
func (a *App) routeHandler(req *http.Request, w http.ResponseWriter) {
	requestPath := req.URL.Path

	//log the request
	var logEntry bytes.Buffer

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
		success := a.tryServingFile(requestPath, req, w)
		if success {
			fmt.Fprintf(&logEntry, "\033[%v;1m%s %s\033[0m", ForeGreen, req.Method, requestPath)
			a.Logger.Print(logEntry.String())
			return
		}
	}

	//Set the default content-type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if !a.filter(w, req) {
		return
	}
	requestPath = req.URL.Path //[SWH|+]

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
				Error(w, 500, "xrsf token error.")

				fmt.Fprintf(&logEntry, "\033[%v;1m%s %s\033[0m", ForeRed, req.Method, requestPath)
				a.Logger.Print(logEntry.String())
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

		if c.App.AppConfig.FormMapToStruct {
			a.StructMap(vc.Elem(), req)
		}

		initM := vc.MethodByName("Init")
		if initM.IsValid() {
			params := []reflect.Value{}
			initM.Call(params)
		}

		//[SWH|+]------------------------------------------Before-Hook
		structName := reflect.ValueOf(route.ctype.String())
		actionName := reflect.ValueOf(route.handler)
		structAction := []reflect.Value{structName, actionName}
		initM = vc.MethodByName("Before")
		if initM.IsValid() {
			if ok := initM.Call(structAction); !ok[0].Bool() {
				return
			}
		}

		ret, err := a.safelyCall(vc, route.handler, args)
		if err != nil {
			c.GetLogger().Println(err)
			//there was an error or panic while calling the handler
			if c.App.AppConfig.Mode == Debug {
				c.Abort(500, err.Error())
			} else if c.App.AppConfig.Mode == Product {
				c.Abort(500, "Server Error")
			}
			fmt.Fprintf(&logEntry, "\033[%v;1m%s %s\033[0m", ForeRed, req.Method, requestPath)
			c.App.Logger.Print(logEntry.String())
			return
		}

		//[SWH|+]------------------------------------------After-Hook
		initM = vc.MethodByName("After")
		if initM.IsValid() {
			actionResult := reflect.ValueOf(ret)
			structAction = []reflect.Value{structName, actionName, actionResult}
			if ok := initM.Call(structAction); !ok[0].Bool() {
				return
			}
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
			fmt.Fprintf(&logEntry, "\033[%v;1m%s %s\033[0m", ForeRed, req.Method, requestPath)
			c.App.Logger.Print(logEntry.String())
			return
		}

		c.SetHeader("Content-Length", strconv.Itoa(len(content)))
		_, err = c.ResponseWriter.Write(content)
		if err != nil {
			a.Logger.Println("Error during write: ", err)
		}

		fmt.Fprintf(&logEntry, "\033[%v;1m%s %s\033[0m", ForeGreen, req.Method, requestPath)
		c.App.Logger.Print(logEntry.String())
		return
	}

	// try serving index.html or index.htm
	if req.Method == "GET" || req.Method == "HEAD" {
		if a.tryServingFile(path.Join(requestPath, "index.html"), req, w) {
			fmt.Fprintf(&logEntry, "\033[%v;1m%s %s\033[0m", ForeGreen, req.Method, requestPath)
			a.Logger.Print(logEntry.String())
			return
		} else if a.tryServingFile(path.Join(requestPath, "index.htm"), req, w) {
			fmt.Fprintf(&logEntry, "\033[%v;1m%s %s\033[0m", ForeGreen, req.Method, requestPath)
			a.Logger.Print(logEntry.String())
			return
		}
	}

	Error(w, 404, "Page not found")

	fmt.Fprintf(&logEntry, "\033[%v;1m%s %s\033[0m", ForeRed, req.Method, requestPath)
	a.Logger.Print(logEntry.String())
}

func (a *App) Error(w http.ResponseWriter, status int, content string) error {
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
func (a *App) safelyCall(vc reflect.Value, method string, args []reflect.Value) (resp []reflect.Value, err error) {
	defer func() {
		if e := recover(); e != nil {
			if !a.Server.Config.RecoverPanic {
				// go back to panic
				panic(e)
			} else {
				resp = nil
				var content string
				content = fmt.Sprintln("Handler crashed with error", e)
				for i := 1; ; i += 1 {
					_, file, line, ok := runtime.Caller(i)
					if !ok {
						break
					}
					content += fmt.Sprintln(file, line)
				}
				a.Logger.Print(content)
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
func (a *App) tryServingFile(name string, req *http.Request, w http.ResponseWriter) bool {
	newPath := name[len(a.BasePath):]
	staticFile := path.Join(a.AppConfig.StaticDir, newPath)
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
	for k, t := range r.Form {
		v := t[0]
		names := strings.Split(k, ".")
		var value reflect.Value = vc
		for i, name := range names {
			name = strings.Title(name)
			if i != len(names)-1 {
				if value.Kind() != reflect.Struct {
					a.Logger.Printf("arg error, value kind is %v\n", value.Kind())
					break
				}

				value = value.FieldByName(name)
				if !value.IsValid() {
					a.Logger.Printf("(%v value is not valid %v)\n", name, value.Interface())
					break
				}
				if !value.CanSet() {
					a.Logger.Printf("can not set %v -> %v\n", name, value.Interface())
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
					a.Logger.Printf("struct %v has no field named %v\n", value, name)
					break
				}
				if !tv.CanSet() {
					a.Logger.Printf("can not set %v\n", k)
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
						a.Logger.Printf("arg %v as int: %v\n", v, err)
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Int64:
					x, err := strconv.ParseInt(v, 10, 64)
					if err != nil {
						a.Logger.Printf("arg %v as int64: %v\n", v, err)
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Float32, reflect.Float64:
					x, err := strconv.ParseFloat(v, 64)
					if err != nil {
						a.Logger.Printf("arg %v as float64: %v\n", v, err)
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					x, err := strconv.ParseUint(v, 10, 64)
					if err != nil {
						a.Logger.Printf("arg %v as uint: %v\n", v, err)
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Struct:
					if tvf, ok := tv.Interface().(FromConversion); ok {
						err := tvf.FromString(v)
						if err != nil {
							a.Logger.Printf("struct %v invoke FromString faild\n", tvf)
						}
					} else if tv.Type().String() == "time.Time" {
						x, err := time.Parse("2006-01-02 15:04:05.000 -0700", v)
						if err != nil {
							x, err = time.Parse("2006-01-02 15:04:05", v)
							if err != nil {
								x, err = time.Parse("2006-01-02", v)
								if err != nil {
									a.Logger.Printf("unsupported time format: %v\n", v)
									break
								}
							}
						}
						l = x
						tv.Set(reflect.ValueOf(l))
					} else {
						a.Logger.Printf("can not set an struct which is not implement Fromconversion interface")
					}
				case reflect.Ptr:
					a.Logger.Printf("can not set an ptr of ptr\n")
				case reflect.Slice, reflect.Array:
					// TODO: currently only support []string, need to
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
							fmt.Println("slice error:", name, err)
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
