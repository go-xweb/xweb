package xweb

import (
	"bytes"
	//"errors"
	"fmt"
	"github.com/astaxie/beego/session"
	"html/template"
	//"io/ioutil"
	"net/http"
	"os"
	"path"
	//"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type App struct {
	BasePath       string
	routes         []route
	filters        []Filter
	Server         *Server
	Config         *AppConfig
	Actions        map[reflect.Type]string
	FuncMaps       map[reflect.Type]template.FuncMap
	SessionManager *session.Manager //Session manager
	RootTemplate   *template.Template
}

type AppConfig struct {
	StaticDirs    map[string]string
	TemplateDir   string
	SessionOn     bool
	MaxUploadSize int64
	CookieSecret  string
}

type route struct {
	r       string
	cr      *regexp.Regexp
	method  string
	handler string
	ctype   reflect.Type
}

func NewApp(path string) *App {
	return &App{BasePath: path,
		Config: &AppConfig{
			StaticDirs:    map[string]string{"static": "static"},
			TemplateDir:   "templates",
			SessionOn:     true,
			MaxUploadSize: 10 * 1024 * 1024,
		},
		Actions:  map[reflect.Type]string{},
		FuncMaps: map[reflect.Type]template.FuncMap{},
	}
}

func (a *App) initApp() {
	if a.Config.SessionOn {
		a.SessionManager, _ = session.NewManager("memory", "beegosessionID", 3600, "")
		go a.SessionManager.GC()
	}
	/*a.RootTemplate = template.New(a.BasePath)
	err := a.initTemplates(a.Config.TemplateDir)
	if err != nil {
		fmt.Printf("initTemplates error: %v\n", err)
		return
	}*/
}

/*func (app *App) walkDir(dir string, f os.FileInfo, err error) error {
	if f == nil {
		return err
	}
	if f.IsDir() {
		//childDir := path.Join(dir, f.Name())
		//fmt.Println(f)
		//if f.Name() != app.Config.TemplateDir {

		//return filepath.Walk(childDir, app.walkDir)
		return nil
	} else if (f.Mode() & os.ModeSymlink) > 0 {
		return nil
	} else {
		//fmt.Println(dir)
		tpath := dir[len(app.Config.TemplateDir):]
		tpath = strings.TrimLeft(tpath, "/")
		//fmt.Println(tpath)
		t := app.RootTemplate.New(tpath)

		var err error
		data, err := ioutil.ReadFile(dir)
		if err != nil {
			fmt.Println(err)
			return err
		}
		content := string(data)
		//fmt.Println(content)
		//_, err = t.Funcs(defaultFuncs).Parse(content)
		//fmt.Printf("%v=%v\n", t, err)
		return err
	}
}

func (a *App) initTemplates(dir string) error {
	if _, err := os.Stat(dir); err != nil {
		return errors.New("dir open err")
	}

	return filepath.Walk(dir, a.walkDir)
}*/

func PathDirName(path string) string {
	d := strings.TrimRight(path, string(os.PathSeparator))
	ps := strings.Split(d, string(os.PathSeparator))
	return ps[len(ps)-1]
}

func (a *App) SetStaticDir(dir string) {
	a.Config.StaticDirs = map[string]string{PathDirName(dir): dir}
}

func (a *App) AddStaticDir(dirs ...string) {
	for _, dir := range dirs {
		a.Config.StaticDirs[PathDirName(dir)] = dir
	}
}

func (a *App) SetTemplateDir(path string) {
	a.Config.TemplateDir = path
}

func (a *App) getTemplatePath(name string) string {
	templateFile := path.Join(a.Config.TemplateDir, name)
	if fileExists(templateFile) {
		return templateFile
	}
	return ""
}

func (app *App) AddAction(cs ...interface{}) {
	for _, c := range cs {
		app.AddRouter(app.BasePath, c)
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

func (a *App) addRoute(r string, method string, t reflect.Type, handler string) {
	cr, err := regexp.Compile(r)
	if err != nil {
		a.Server.Logger.Printf("Error in route regex %q\n", r)
		return
	}

	a.routes = append(a.routes, route{r, cr, method, handler, t})
}

func (app *App) AddRouter(url string, c interface{}) {
	t := reflect.TypeOf(c).Elem()
	app.Actions[t] = url
	app.FuncMaps[t] = defaultFuncs
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag
		tagStr := tag.Get("xweb")
		if tagStr != "" {
			a := strings.Title(t.Field(i).Name)
			v := reflect.ValueOf(c).MethodByName(a)
			if v.IsValid() {
				tags := strings.Split(tagStr, " ")
				methods := []string{"GET", "POST"}
				path := tagStr
				if len(tags) >= 2 {
					methods = strings.Split(tags[0], "|")
					path = tags[1]
				}
				for _, method := range methods {
					app.addRoute(strings.TrimRight(url, "/")+path, strings.ToUpper(method), t, a)
				}
			}
		} else {
			if t.Field(i).Type == reflect.TypeOf(Mapper{}) {
				name := t.Field(i).Name
				a := strings.Title(name)
				v := reflect.ValueOf(c).MethodByName(a)
				if v.IsValid() {
					app.addRoute(strings.TrimRight(url, "/")+"/"+name, "GET", t, a)
					app.addRoute(strings.TrimRight(url, "/")+"/"+name, "POST", t, a)
				}
			}
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
	req.ParseForm()
	req.ParseMultipartForm(a.Config.MaxUploadSize)

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
		if req.Method != route.method && !(req.Method == "HEAD" && route.method == "GET") {
			continue
		}

		if !cr.MatchString(requestPath) {
			continue
		}
		match := cr.FindStringSubmatch(requestPath)

		if len(match[0]) != len(requestPath) {
			continue
		}

		var args []reflect.Value
		for _, arg := range match[1:] {
			args = append(args, reflect.ValueOf(arg))
		}
		vc := reflect.New(route.ctype)
		c := Action{Request: req, App: a,
			ResponseWriter: w, BasePath: strings.TrimRight(a.BasePath, "/") + a.Actions[route.ctype]}
		fieldA := vc.Elem().FieldByName("Action")
		if fieldA.IsValid() {
			fieldA.Set(reflect.ValueOf(c))
		}

		a.StructMap(vc.Elem(), req)
		fieldC := vc.Elem().FieldByName("C")
		if fieldC.IsValid() {
			fieldC.Set(reflect.ValueOf(vc))
		}

		initM := vc.MethodByName("Init")
		if initM.IsValid() {
			params := []reflect.Value{}
			initM.Call(params)
		}

		ret, err := a.safelyCall(vc, route.handler, args)
		if err != nil {
			//there was an error or panic while calling the handler
			c.Abort(500, "Server Error")
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

/*func (a *App) StaticUrl(url string) string {
	us := strings.Split(url, "/")
	path := a.Config.StaticDirs[us[0]] + strings.Join(us[1:], "/")
	return a.BasePath + url + "?v="
}*/

//func (a *App) xsrf_form_html() string {

//}

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
	paths := strings.Split(newPath, "/")
	for _, dir := range a.Config.StaticDirs {
		dirs := strings.Split(dir, "/")
		if dirs[len(dirs)-1] == paths[0] {
			staticFile := path.Join(dir, strings.Join(paths[1:], "/"))
			if fileExists(staticFile) {
				http.ServeFile(w, req, staticFile)
				return true
			}
		}
	}
	//fmt.Println(name)
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
					a.Server.Logger.Printf("struct %v has no field named %v", value, name)
					break
				}
				if !tv.CanSet() {
					a.Server.Logger.Printf("can not set ", k)
					break
				}
				var l interface{}
				switch k := tv.Kind(); k {
				case reflect.String:
					l = v
					tv.Set(reflect.ValueOf(l))
				case reflect.Bool:
					l = (v == "true")
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
