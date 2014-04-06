package xweb

import (
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/howeyc/fsnotify"
)

func IsNil(a interface{}) bool {
	switch a.(type) {
	case nil:
		return true
	}
	return false
}

func Add(left interface{}, right interface{}) interface{} {
	var rleft, rright int64
	var fleft, fright float64
	var isInt bool = true
	switch left.(type) {
	case int:
		rleft = int64(left.(int))
	case int8:
		rleft = int64(left.(int8))
	case int16:
		rleft = int64(left.(int16))
	case int32:
		rleft = int64(left.(int32))
	case int64:
		rleft = left.(int64)
	case float32:
		fleft = float64(left.(float32))
		isInt = false
	case float64:
		fleft = left.(float64)
		isInt = false
	}

	switch right.(type) {
	case int:
		rright = int64(right.(int))
	case int8:
		rright = int64(right.(int8))
	case int16:
		rright = int64(right.(int16))
	case int32:
		rright = int64(right.(int32))
	case int64:
		rright = right.(int64)
	case float32:
		fright = float64(left.(float32))
		isInt = false
	case float64:
		fleft = left.(float64)
		isInt = false
	}

	var intSum int64 = rleft + rright

	if isInt {
		return intSum
	} else {
		return fleft + fright + float64(intSum)
	}
}

func Subtract(left interface{}, right interface{}) interface{} {
	var rleft, rright int64
	var fleft, fright float64
	var isInt bool = true
	switch left.(type) {
	case int:
		rleft = int64(left.(int))
	case int8:
		rleft = int64(left.(int8))
	case int16:
		rleft = int64(left.(int16))
	case int32:
		rleft = int64(left.(int32))
	case int64:
		rleft = left.(int64)
	case float32:
		fleft = float64(left.(float32))
		isInt = false
	case float64:
		fleft = left.(float64)
		isInt = false
	}

	switch right.(type) {
	case int:
		rright = int64(right.(int))
	case int8:
		rright = int64(right.(int8))
	case int16:
		rright = int64(right.(int16))
	case int32:
		rright = int64(right.(int32))
	case int64:
		rright = right.(int64)
	case float32:
		fright = float64(left.(float32))
		isInt = false
	case float64:
		fleft = left.(float64)
		isInt = false
	}

	if isInt {
		return rleft - rright
	} else {
		return fleft + float64(rleft) - (fright + float64(rright))
	}
}

func Now() time.Time {
	return time.Now()
}

func FormatDate(t time.Time, format string) string {
	return t.Format(format)
}

func Eq(left interface{}, right interface{}) bool {
	leftIsNil := (left == nil)
	rightIsNil := (right == nil)
	if leftIsNil || rightIsNil {
		if leftIsNil && rightIsNil {
			return true
		}
		return false
	}
	return left == right
}

func Html(raw string) template.HTML {
	return template.HTML(raw)
}

//Usage:UrlFor("main:root:/user/login") or UrlFor("root:/user/login") or UrlFor("/user/login") or UrlFor()
func UrlFor(args ...string) string {
	s := [3]string{"main", "root", ""}
	var u []string
	size := len(args)
	if size > 0 {
		u = strings.Split(args[0], ":")
	} else {
		u = []string{""}
	}
	var appUrl string = ""
	switch len(u) {
	case 1:
		s[2] = u[0]
	case 2:
		s[1] = u[0]
		s[2] = u[1]
	default:
		s[0] = u[0]
		s[1] = u[1]
		s[2] = u[2]
	}
	var url, prefix, suffix string
	if server, ok := Servers[s[0]]; ok {
		url += server.Config.Url
		prefix = server.Config.UrlPrefix
		suffix = server.Config.UrlSuffix
		if appPath, ok := server.AppName[s[1]]; ok {
			appUrl = appPath
		}
	}
	url = strings.TrimRight(url, "/") + "/"
	if size == 0 {
		return url
	}
	if appUrl != "/" {
		appUrl = strings.TrimLeft(appUrl, "/")
		if length := len(appUrl); length > 0 && appUrl[length-1] != '/' {
			appUrl = appUrl + "/"
		}
	} else {
		appUrl = ""
	}
	url += prefix + appUrl
	if s[2] == "" {
		return url
	}
	url += strings.TrimLeft(s[2], "/") + suffix
	return url
}

var (
	defaultFuncs template.FuncMap = template.FuncMap{
		"Now":        Now,
		"Eq":         Eq,
		"FormatDate": FormatDate,
		"Html":       Html,
		"Add":        Add,
		"Subtract":   Subtract,
		"IsNil":      IsNil,
		"UrlFor":     UrlFor,
	}
)

type TemplateMgr struct {
	Caches   map[string][]byte
	mutex    *sync.Mutex
	RootDir  string
	Ignores  map[string]bool
	IsReload bool
	app      *App
}

func (self *TemplateMgr) Moniter(rootDir string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	done := make(chan bool)
	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				if ev == nil {
					break
				}
				if _, ok := self.Ignores[filepath.Base(ev.Name)]; ok {
					break
				}
				d, err := os.Stat(ev.Name)
				if err != nil {
					break
				}

				if ev.IsCreate() {
					if d.IsDir() {
						watcher.Watch(ev.Name)
					} else {
						tmpl := ev.Name[len(self.RootDir)+1:]
						content, err := ioutil.ReadFile(ev.Name)
						if err != nil {
							self.app.Error("loaded template %v failed: %v", tmpl, err)
							break
						}
						self.app.Info("loaded template file %v success", tmpl)
						self.CacheTemplate(tmpl, content)
					}
				} else if ev.IsDelete() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						tmpl := ev.Name[len(self.RootDir)+1:]
						self.CacheDelete(tmpl)
					}
				} else if ev.IsModify() {
					if d.IsDir() {
					} else {
						tmpl := ev.Name[len(self.RootDir)+1:]
						content, err := ioutil.ReadFile(ev.Name)
						if err != nil {
							self.app.Error("reloaded template %v failed: %v", tmpl, err)
							break
						}

						self.CacheTemplate(tmpl, content)
						self.app.Info("reloaded template %v success", tmpl)
					}
				} else if ev.IsRename() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						tmpl := ev.Name[len(self.RootDir)+1:]
						self.CacheDelete(tmpl)
					}
				}
			case err := <-watcher.Error:
				self.app.Error("error: %v", err)
			}
		}
	}()

	err = filepath.Walk(self.RootDir, func(f string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return watcher.Watch(f)
		}
		return nil
	})

	if err != nil {
		self.app.Error(err.Error())
		return err
	}

	<-done

	watcher.Close()
	return nil
}

func (self *TemplateMgr) CacheAll(rootDir string) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	err := filepath.Walk(rootDir, func(f string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		tmpl := f[len(rootDir)+1:]
		tmpl = strings.Replace(tmpl, "\\", "/", -1) //[SWH|+]fix windows env
		if _, ok := self.Ignores[filepath.Base(tmpl)]; !ok {
			fpath := filepath.Join(self.RootDir, tmpl)
			content, err := ioutil.ReadFile(fpath)
			if err != nil {
				self.app.Debug("Load template %s error: %v", fpath, err)
				return err
			}
			self.app.Debug("Loaded template %s", fpath)
			self.Caches[tmpl] = content
		}
		return nil
	})
	return err
}

func (self *TemplateMgr) Init(app *App, rootDir string, reload bool) error {
	self.RootDir = rootDir
	self.Caches = make(map[string][]byte)
	self.mutex = &sync.Mutex{}
	self.app = app
	if dirExists(rootDir) {
		self.CacheAll(rootDir)

		if reload {
			go self.Moniter(rootDir)
		}
	}

	return nil
}

func (self *TemplateMgr) GetTemplate(tmpl string) ([]byte, error) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if content, ok := self.Caches[tmpl]; ok {
		self.app.Debug("load template %v from cache", tmpl)
		return content, nil
	}

	content, err := ioutil.ReadFile(path.Join(self.RootDir, tmpl))
	if err == nil {
		self.app.Debug("load template %v from the file:", tmpl)
		self.Caches[tmpl] = content
	}
	return content, err
}

func (self *TemplateMgr) CacheTemplate(tmpl string, content []byte) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	tmpl = strings.Replace(tmpl, "\\", "/", -1)
	self.app.Debug("Update template %v on cache", tmpl)
	self.Caches[tmpl] = content
	return
}

func (self *TemplateMgr) CacheDelete(tmpl string) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	tmpl = strings.Replace(tmpl, "\\", "/", -1)
	self.app.Debug("Delete template %v from cache", tmpl)
	delete(self.Caches, tmpl)
	return
}
