package xweb

import (
	"reflect"
	"regexp"
	"strings"
)

type Route struct {
	Path           string          //path string
	CompiledRegexp *regexp.Regexp  //path regexp
	HttpMethods    map[string]bool //GET POST HEAD DELETE etc.
	HandlerMethod  string          //struct method name
	HandlerElement reflect.Type    //handler element
	hasAction      bool
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
		} else {
			app.Logger.Warn("AutoAction needs a named ends with Action")
		}
	}
}

func (a *App) addRoute(r string, methods map[string]bool,
	t reflect.Type, handler string, hasAction bool) {
	cr, err := regexp.Compile(r)
	if err != nil {
		a.Logger.Errorf("Error in route regex %q: %s", r, err)
		return
	}
	a.Routes = append(a.Routes, Route{
		Path:           r,
		CompiledRegexp: cr,
		HttpMethods:    methods,
		HandlerMethod:  handler,
		HandlerElement: t,
		hasAction:      hasAction,
	})
}

func (a *App) addEqRoute(r string, methods map[string]bool,
	t reflect.Type, handler string, hasAction bool) {
	if _, ok := a.RoutesEq[r]; !ok {
		a.RoutesEq[r] = make(map[string]Route)
	}
	for v, _ := range methods {
		a.RoutesEq[r][v] = Route{
			HandlerMethod:  handler,
			HandlerElement: t,
			hasAction:      hasAction,
		}
	}
}

var (
	mapperType = reflect.TypeOf(Mapper{})
)

func (app *App) AddRouter(url string, c interface{}) {
	vc := reflect.ValueOf(c)
	t := vc.Type().Elem()
	app.ActionsPath[t] = url
	app.Actions[t.Name()] = c
	app.ActionsNamePath[t.Name()] = url

	hasAction := vc.Elem().FieldByName("Action").IsValid()

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
			app.addEqRoute(removeStick(p), methods, t, a, hasAction)
		} else {
			app.addRoute(removeStick(p), methods, t, a, hasAction)
		}
	}

	// added a default method as /
	v := reflect.ValueOf(c).MethodByName("Execute")
	if !v.IsValid() {
		return
	}
	p := strings.TrimRight(url, "/") + "/"
	methods := map[string]bool{"GET": true, "POST": true}
	app.addEqRoute(removeStick(p), methods, t, "Execute", hasAction)
}

func (a *App) findRoute(reqPath, allowMethod string) (Route, []reflect.Value, bool) {
	var route Route
	var isFind bool
	var args = make([]reflect.Value, 0)
	if routes, ok := a.RoutesEq[reqPath]; ok {
		if route, ok = routes[allowMethod]; ok {
			isFind = true
		}
	}

	if !isFind {
		for _, route = range a.Routes {
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

			for _, arg := range match[1:] {
				args = append(args, reflect.ValueOf(arg))
			}
			isFind = true
			break
		}
	}

	return route, args, isFind
}
