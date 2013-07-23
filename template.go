package xweb

import (
	"fmt"
	"github.com/howeyc/fsnotify"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

func Add(left int, right int) int {
	return left + right
}

func FormatDate(t time.Time, format string) string {
	return t.Format(format)
}

func Eq(left interface{}, right interface{}) bool {
	return left == right
}

func Html(raw string) template.HTML {
	return template.HTML(raw)
}

var (
	defaultFuncs template.FuncMap = template.FuncMap{
		"Eq":         Eq,
		"FormatDate": FormatDate,
		"Html":       Html,
		"Add":        Add,
	}
)

type TemplateMgr struct {
	Caches   map[string][]byte
	mutex    *sync.Mutex
	RootDir  string
	Ignores  map[string]bool
	IsReload bool
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
						self.Caches[tmpl] = content
						self.mutex.Unlock()
					}
				} else if ev.IsDelete() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						self.mutex.Lock()
						delete(self.Caches, ev.Name[len(self.RootDir)+1:])
						self.mutex.Unlock()
					}
				} else if ev.IsModify() {
					if d.IsDir() {
					} else {
						tmpl := ev.Name[len(self.RootDir)+1:]
						content, err := ioutil.ReadFile(ev.Name)
						if err != nil {
							break
						}

						self.mutex.Lock()
						self.Caches[tmpl] = content
						self.mutex.Unlock()
					}
				} else if ev.IsRename() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						self.mutex.Lock()
						delete(self.Caches, ev.Name[len(self.RootDir)+1:])
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

func (self *TemplateMgr) Init(rootDir string, reload bool) error {
	self.RootDir = rootDir
	self.Caches = make(map[string][]byte)
	self.mutex = &sync.Mutex{}

	self.CacheAll(rootDir)

	if reload {
		go self.Moniter(rootDir)
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
