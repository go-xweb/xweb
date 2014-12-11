package xweb

import (
	"reflect"
	"regexp"
	"strings"
)

// Route
type Route struct {
	Path           string          //path string
	CompiledRegexp *regexp.Regexp  //path regexp
	HttpMethods    map[string]bool //GET POST HEAD DELETE etc.
	HandlerMethod  string          //struct method name
	HandlerElement reflect.Type    //handler element
	hasAction      bool
	method         reflect.Value
	isStruct       bool // is this route is a struct or a func
	isPtr          bool // when use struct, is the first receiver is ptr or struct
}

func (route *Route) IsStruct() bool {
	return route.isStruct
}

func (route *Route) newAction() reflect.Value {
	if route.isStruct {
		vc := reflect.New(route.HandlerElement)

		if route.hasAction {
			c := &Action{
				C: vc,
			}

			vc.Elem().FieldByName("Action").Set(reflect.ValueOf(c))
		}
		return vc
	} else {
		return route.method
	}
}

type Router struct {
	basePath        string
	Routes          []*Route
	RoutesEq        map[string]map[string]*Route
	Actions         map[string]interface{}
	ActionsPath     map[reflect.Type]string
	ActionsNamePath map[string]string
}

func NewRouter(basePath string) *Router {
	return &Router{
		basePath:        basePath,
		Routes:          make([]*Route, 0),
		RoutesEq:        make(map[string]map[string]*Route),
		Actions:         map[string]interface{}{},
		ActionsPath:     map[reflect.Type]string{},
		ActionsNamePath: map[string]string{},
	}
}

func (router *Router) Action(name string) interface{} {
	if v, ok := router.Actions[name]; ok {
		return v
	}
	return nil
}

func (router *Router) AddAction(cs ...interface{}) {
	for _, c := range cs {
		router.AddRouter(router.basePath, c)
	}
}

func (router *Router) AutoAction(cs ...interface{}) {
	for _, c := range cs {
		t := reflect.Indirect(reflect.ValueOf(c)).Type()
		name := t.Name()
		if strings.HasSuffix(name, "Action") {
			name = strings.ToLower(name[:len(name)-6])
		}
		router.AddRouter(JoinPath(router.basePath, name), c)
	}
}

func (router *Router) addRoute(r string, methods map[string]bool,
	t reflect.Type, handler string, hasAction bool,
	method reflect.Value, isStruct, isPtr bool) error {
	cr, err := regexp.Compile(r)
	if err != nil {
		return err
	}
	router.Routes = append(router.Routes, &Route{
		Path:           r,
		CompiledRegexp: cr,
		HttpMethods:    methods,
		HandlerMethod:  handler,
		HandlerElement: t,
		hasAction:      hasAction,
		method:         method,
		isStruct:       isStruct,
		isPtr:          isPtr,
	})
	return nil
}

func (router *Router) addEqRoute(r string, methods map[string]bool,
	t reflect.Type, handler string, hasAction bool,
	method reflect.Value, isStruct, isPtr bool) {
	if _, ok := router.RoutesEq[r]; !ok {
		router.RoutesEq[r] = make(map[string]*Route)
	}
	for v, _ := range methods {
		router.RoutesEq[r][v] = &Route{
			HandlerMethod:  handler,
			HandlerElement: t,
			hasAction:      hasAction,
			method:         method,
			isStruct:       isStruct,
			isPtr:          isPtr,
		}
	}
}

var (
	mapperType = reflect.TypeOf(Mapper{})
)

func (router *Router) AddRouter(url string, c interface{}) {
	vc := reflect.ValueOf(c)
	if vc.Kind() == reflect.Func {
		router.addFuncRouter(url, c)
	} else if vc.Kind() == reflect.Ptr && vc.Elem().Kind() == reflect.Struct {
		router.addStructRouter(url, c)
	}
}

func (router *Router) addFuncRouter(url string, c interface{}) {
	vc := reflect.ValueOf(c)
	t := vc.Type()
	methods := map[string]bool{"GET": true, "POST": true}
	router.addEqRoute(removeStick(url), methods, t, "", false, vc, false, false)
}

func (router *Router) addStructRouter(url string, c interface{}) {
	vc := reflect.ValueOf(c)
	t := vc.Type().Elem()
	router.ActionsPath[t] = url
	router.Actions[t.Name()] = c
	router.ActionsNamePath[t.Name()] = url

	hasAction := vc.Elem().FieldByName("Action").IsValid()

	var usedFuncNames = make(map[string]bool)

	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Type != mapperType {
			continue
		}
		name := t.Field(i).Name
		a := strings.Title(name)
		var m reflect.Method
		var ok bool
		if m, ok = t.MethodByName(a); !ok {
			continue
		}
		usedFuncNames[a] = true

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
			router.addEqRoute(removeStick(p), methods,
				t, a, hasAction, m.Func, true, false)
		} else {
			router.addRoute(removeStick(p), methods,
				t, a, hasAction, m.Func, true, false)
		}
	}

	// if method Do has been used, so don't mapping
	if _, ok := usedFuncNames["Do"]; ok {
		return
	}

	// added a default method Do as /
	var m reflect.Method
	var ok bool
	if m, ok = t.MethodByName("Do"); !ok {
		return
	}

	var isPtr = (m.Type.In(0).Kind() == reflect.Ptr)

	p := strings.TrimRight(url, "/") + "/"
	methods := map[string]bool{"GET": true, "POST": true}
	router.addEqRoute(removeStick(p), methods, t, "Do",
		hasAction, m.Func, true, isPtr)
}

// when a request ask, then match the correct route
func (router *Router) Match(reqPath, allowMethod string) (*Route, []reflect.Value) {
	var route *Route
	var args = make([]reflect.Value, 0)

	// for non-regular path, search the map
	if routes, ok := router.RoutesEq[reqPath]; ok {
		if route, ok = routes[allowMethod]; ok {
			return route, args
		}
	}

	for _, r := range router.Routes {
		cr := r.CompiledRegexp

		//if the methods don't match, skip this handler (except HEAD can be used in place of GET)
		if _, ok := r.HttpMethods[allowMethod]; !ok {
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

		return route, args
	}

	return nil, nil
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
func (router *Router) Nodes() (r map[string]map[string][]string) {
	r = make(map[string]map[string][]string)
	for _, val := range router.Routes {
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
	for _, vals := range router.RoutesEq {
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
