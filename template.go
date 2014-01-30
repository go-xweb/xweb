package xweb

import (
	"fmt"
	"github.com/howeyc/fsnotify"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
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

var (
	defaultFuncs template.FuncMap = template.FuncMap{
		"Now":        Now,
		"Eq":         Eq,
		"FormatDate": FormatDate,
		"Html":       Html,
		"Add":        Add,
		"Subtract":   Subtract,
		"IsNil":      IsNil,
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
							break
						}

						self.mutex.Lock()
						tmpl = strings.Replace(tmpl, "\\", "/", -1) //[SWH|+]fix windows env
						self.Caches[tmpl] = content
						self.mutex.Unlock()
					}
				} else if ev.IsDelete() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						self.mutex.Lock()
						tmpl := ev.Name[len(self.RootDir)+1:]
						tmpl = strings.Replace(tmpl, "\\", "/", -1) //[SWH|+]fix windows env
						delete(self.Caches, tmpl)
						self.mutex.Unlock()
					}
				} else if ev.IsModify() {
					if d.IsDir() {
					} else {
						tmpl := ev.Name[len(self.RootDir)+1:]
						content, err := ioutil.ReadFile(ev.Name)
						if err != nil {
							self.app.Logger.Println("reloaded template", tmpl, "failed")
							break
						}

						self.mutex.Lock()
						tmpl = strings.Replace(tmpl, "\\", "/", -1) //[SWH|+]fix windows env
						self.Caches[tmpl] = content
						self.mutex.Unlock()
						self.app.Logger.Println("reloaded template", tmpl)
					}
				} else if ev.IsRename() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						self.mutex.Lock()
						tmpl := ev.Name[len(self.RootDir)+1:]
						tmpl = strings.Replace(tmpl, "\\", "/", -1) //[SWH|+]fix windows env
						delete(self.Caches, tmpl)
						self.mutex.Unlock()
					}
				}
			case err := <-watcher.Error:
				fmt.Println("error:", err)
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
		fmt.Println(err)
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
			content, err := ioutil.ReadFile(path.Join(self.RootDir, tmpl))
			if err != nil {
				return err
			}
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
		return content, nil
	}

	content, err := ioutil.ReadFile(path.Join(self.RootDir, tmpl))
	if err == nil {
		self.Caches[tmpl] = content
	}
	return content, err
}
