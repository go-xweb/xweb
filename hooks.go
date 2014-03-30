/************************
[钩子引擎 (version 0.3)]
@author:S.W.H
@E-mail:swh@admpub.com
@update:2014-01-18
************************/
package xweb

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrParamsNotAdapted             = errors.New("The number of params is not adapted.")
	XHook               *HookEngine = NewHookEngine(10)
)

type Hook []reflect.Value
type HookEngine struct {
	Hooks map[string]Hook
	Index map[string]uint
}

func (f *HookEngine) Bind(name string, fns ...interface{}) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = errors.New(name + " is not callable.")
		}
	}()
	if _, ok := f.Hooks[name]; !ok {
		f.Hooks[name] = make(Hook, 0)
	}
	if _, ok := f.Index[name]; !ok {
		f.Index[name] = 0
	}
	hln := uint(len(f.Hooks[name]))
	fln := f.Index[name] + 1 + uint(len(fns))
	if hln < fln {
		for _, fn := range fns {
			v := reflect.ValueOf(fn)
			f.Hooks[name] = append(f.Hooks[name], v)
			f.Index[name]++
		}
	} else {
		for _, fn := range fns {
			v := reflect.ValueOf(fn)
			f.Hooks[name][f.Index[name]] = v
			f.Index[name]++
		}
	}
	return
}

func (f *HookEngine) Call(name string, params ...interface{}) (result []reflect.Value, err error) {
	if _, ok := f.Hooks[name]; !ok {
		err = errors.New(name + " does not exist.")
		return
	}
	ln := len(params)
	in := make([]reflect.Value, ln)
	for k, param := range params {
		in[k] = reflect.ValueOf(param)
	}
	for _, v := range f.Hooks[name] {
		if v.IsValid() == false {
			continue
		}
		if ln != v.Type().NumIn() {
			continue
			err = ErrParamsNotAdapted
			return
		}
		result = v.Call(in)
		for _k, _v := range result {
			in[_k] = _v
		}
	}
	if len(result) == 0 {
		err = errors.New(name + " have nothing to do.")
	}
	return
}

func (f *HookEngine) Value(c []reflect.Value, index int) (r interface{}) {
	if len(c) >= index && c[index].CanInterface() {
		r = c[index].Interface()
	}
	return
}

func (f *HookEngine) String(c reflect.Value) string {
	return fmt.Sprintf("%s", c)
}

func NewHookEngine(size int) *HookEngine {
	h := &HookEngine{Hooks: make(map[string]Hook, size), Index: make(map[string]uint, size)}

	//func(mux *http.ServeMux) *http.ServeMux
	h.Hooks["MuxHandle"] = make(Hook, 0)

	//func(result *bool, serv *Server, w http.ResponseWriter, req *http.Request) *bool
	h.Hooks["BeforeProcess"] = make(Hook, 0)

	//func(result *bool, serv *Server, w http.ResponseWriter, req *http.Request) *bool
	h.Hooks["AfterProcess"] = make(Hook, 0)

	//func(content string, action *Action) string
	h.Hooks["BeforeRender"] = make(Hook, 0)

	//func(content []byte, action *Action) []byte
	h.Hooks["AfterRender"] = make(Hook, 0)
	return h
}
