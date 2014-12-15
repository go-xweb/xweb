package xweb

import (
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-xweb/httpsession"
	"github.com/lunny/tango"
	"github.com/tango-contrib/bind"
	"github.com/tango-contrib/xsrf"
)

type App struct {
	*tango.Tango
	*Router
	*Configs
	*tango.Render

	BasePath string
	Name     string //[SWH|+]

	Server    *Server
	AppConfig AppConfig
	Config    map[string]interface{}

	// TODO: refactoring this
	SessionManager *httpsession.Manager //Session manager
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
		Tango:   tango.New(),
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
	a.Map(logger)

	a.Use(
		a.Configs,
		tango.NewLogging(logger),
		tango.NewRecovery(a.AppConfig.Mode == Debug),
	)

	if a.Server.Config.EnableGzip {
		a.Use(tango.NewCompress(a.Server.Config.StaticExtensionsToGzip))
	}

	a.Use(
		tango.HandlerFunc(tango.ReturnHandler),
		NewEventsHandle(),
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
		a.FuncMaps["StaticUrl"] = func(url string) string {
			return path.Join(a.basePath, url)
		}
	}

	render := tango.NewRender(
		a.AppConfig.TemplateDir,
		a.AppConfig.ReloadTemplates,
		a.AppConfig.CacheTemplates,
	)
	a.Render = render
	a.Use(render)

	if a.AppConfig.CheckXsrf {
		a.Use(xsrf.NewXsrf(a.AppConfig.SessionTimeout))
	}

	if a.AppConfig.SessionOn {
		a.Use(tango.NewSessions(
			a.AppConfig.SessionTimeout,
		))
	}
}

func (a *App) SetStaticDir(dir string) {
	a.AppConfig.StaticDir = dir
}

func (a *App) SetTemplateDir(path string) {
	a.AppConfig.TemplateDir = path
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
	tm := time.Now().UTC()
	w.Header().Set("Date", webTime(tm))

	//Set the default content-type
	//w.Header().Set("Content-Type", "text/html; charset=utf-8")

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
