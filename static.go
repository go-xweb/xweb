package xweb

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/howeyc/fsnotify"
)

type StaticVerMgr struct {
	Caches  map[string]string
	mutex   *sync.Mutex
	Path    string
	Ignores map[string]bool
	app     *App
}

func (self *StaticVerMgr) Moniter(staticPath string) error {
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
						url := ev.Name[len(self.Path)+1:]
						ver := self.getFileVer(url)
						if ver != "" {
							self.mutex.Lock()
							self.Caches[url] = ver
							self.mutex.Unlock()
							self.app.Infof("static file %s is created.", url)
						}
					}
				} else if ev.IsDelete() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						pa := ev.Name[len(self.Path)+1:]
						self.mutex.Lock()
						delete(self.Caches, pa)
						self.mutex.Unlock()
						self.app.Infof("static file %s is deleted.", pa)
					}
				} else if ev.IsModify() {
					if d.IsDir() {
					} else {
						url := ev.Name[len(staticPath)+1:]
						ver := self.getFileVer(url)
						if ver != "" {
							self.mutex.Lock()
							self.Caches[url] = ver
							self.mutex.Unlock()
							self.app.Infof("static file %s is reloaded.", url)
						}
					}
				} else if ev.IsRename() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						url := ev.Name[len(staticPath)+1:]
						self.mutex.Lock()
						delete(self.Caches, url)
						self.mutex.Unlock()
						self.app.Infof("static file %s is deleted.\n", url)
					}
				}
			case err := <-watcher.Error:
				self.app.Errorf("error: %v", err)
			}
		}
	}()

	err = filepath.Walk(staticPath, func(f string, info os.FileInfo, err error) error {
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

func (self *StaticVerMgr) Init(app *App, staticPath string) error {
	self.Path = staticPath
	self.Caches = make(map[string]string)
	self.mutex = &sync.Mutex{}
	self.Ignores = map[string]bool{".DS_Store": true}
	self.app = app

	if dirExists(staticPath) {
		self.CacheAll(staticPath)

		go self.Moniter(staticPath)
	}

	return nil
}

func (self *StaticVerMgr) getFileVer(url string) string {
	//content, err := ioutil.ReadFile(path.Join(self.Path, url))
	fPath := filepath.Join(self.Path, url)
	self.app.Debug("loaded static ", fPath)
	f, err := os.Open(fPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	fInfo, err := f.Stat()
	if err != nil {
		return ""
	}

	content := make([]byte, int(fInfo.Size()))
	_, err = f.Read(content)
	if err == nil {
		h := md5.New()
		io.WriteString(h, string(content))
		return fmt.Sprintf("%x", h.Sum(nil))[0:4]
	}
	return ""
}

func (self *StaticVerMgr) CacheAll(staticPath string) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	err := filepath.Walk(staticPath, func(f string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		rp := f[len(staticPath)+1:]
		if _, ok := self.Ignores[filepath.Base(rp)]; !ok {
			self.Caches[rp] = self.getFileVer(rp)
		}
		return nil
	})
	return err
}

func (self *StaticVerMgr) GetVersion(url string) string {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if ver, ok := self.Caches[url]; ok {
		return ver
	}

	ver := self.getFileVer(url)
	if ver != "" {
		self.Caches[url] = ver
	}
	return ver
}
