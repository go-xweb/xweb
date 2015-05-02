package xweb

import (
	"html/template"
	"net/http"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/lunny/tango"
	"github.com/tango-contrib/bind"
	"github.com/tango-contrib/renders"
	"github.com/tango-contrib/session"
	"github.com/tango-contrib/xsrf"
)

type App struct {
	*tango.Tango
	*Router
	*Configs

	data     renders.T
	funcs    template.FuncMap
	sessions *session.Sessions

	BasePath string
	Name     string //[SWH|+]

	Server    *Server
	AppConfig AppConfig
	Config    map[string]interface{}
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

	t := tango.New()
	return &App{
		Tango:   t,
		Router:  NewRouter(basePath),
		Configs: NewConfigs(),

		BasePath:  basePath,
		Name:      name,
		AppConfig: DefaultAppConfig,
	}
}

func (a *App) Run(addr string) {
	if a.Server == nil {
		a.Server = mainServer
		a.Server.AddApp(a)
	}
	a.Server.Run(addr)
}

func (a *App) initApp() {
	// TODO: should test if logger has been mapped
	logger := a.Server.Logger

	a.Use(
		a.Configs,
		tango.Logging(),
		tango.Recovery(a.AppConfig.Mode == Debug),
	)

	if a.Server.Config.EnableGzip {
		a.Use(tango.Compresses(a.Server.Config.StaticExtensionsToGzip))
	}

	a.Use(
		tango.Return(),
		&Actions{},
		&Requests{},
		a,
	)

	if a.AppConfig.FormMapToStruct {
		a.Use(&bind.Binds{})
	}

	if a.AppConfig.StaticFileVersion {
		a.Use(NewStaticVersions(
			logger,
			a.AppConfig.StaticDir,
			a.basePath))
	} else {
		// even if don't use static file version, is still
		a.funcs["StaticUrl"] = func(url string) string {
			return path.Join(a.basePath, url)
		}
	}

	a.Use(renders.New(renders.Options{
		Directory: a.AppConfig.TemplateDir,
		Reload:    a.AppConfig.ReloadTemplates,
		//a.AppConfig.CacheTemplates,
		Funcs: a.funcs,
		Vars:  a.data,
	}))

	if a.AppConfig.CheckXsrf {
		a.Use(xsrf.New(a.AppConfig.SessionTimeout))
	}

	if a.AppConfig.SessionOn {
		if a.sessions == nil {
			a.sessions = session.New(session.Options{
				MaxAge: a.AppConfig.SessionTimeout,
			})
		}
		a.Use(a.sessions)
	}
	a.Use(Events())
}

func (a *App) SetStaticDir(dir string) {
	a.AppConfig.StaticDir = dir
}

func (a *App) SetTemplateDir(path string) {
	a.AppConfig.TemplateDir = path
}

func (a *App) AddTmplVar(name string, varOrFun interface{}) {
	v := reflect.ValueOf(varOrFun)
	if v.Type().Kind() == reflect.Func {
		a.funcs[name] = varOrFun
	} else {
		a.data[name] = varOrFun
	}
}

func (a *App) AddTmplVars(t *T) {
	for name, v := range *t {
		a.AddTmplVar(name, v)
	}
}

func (a *App) ServeHttp(w http.ResponseWriter, req *http.Request) {
	//ignore errors from ParseForm because it's usually harmless.
	ct := req.Header.Get("Content-Type")
	if strings.Contains(ct, "multipart/form-data") {
		req.ParseMultipartForm(a.AppConfig.MaxUploadSize)
	} else {
		req.ParseForm()
	}

	//set some default headers
	w.Header().Set("Server", "xweb")
	w.Header().Set("Date", webTime(time.Now().UTC()))

	a.Tango.ServeHTTP(w, req)
}

type AppHandler interface {
	SetApp(*App)
}

func (app *App) Handle(ctx *tango.Context) {
	if action := ctx.Action(); action != nil {
		if apper, ok := action.(AppHandler); ok {
			apper.SetApp(app)
		}
	}

	ctx.Next()
}
