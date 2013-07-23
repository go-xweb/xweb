package xweb

import (
	"crypto/md5"
	"fmt"
	"github.com/howeyc/fsnotify"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
)

type StaticVerMgr struct {
	Caches  map[string]string
	mutex   *sync.Mutex
	Path    string
	Ignores map[string]bool
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
						}
					}
				} else if ev.IsDelete() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						self.mutex.Lock()
						delete(self.Caches, ev.Name[len(self.Path)+1:])
						self.mutex.Unlock()
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
						}
					}
				} else if ev.IsRename() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						self.mutex.Lock()
						delete(self.Caches, ev.Name[len(staticPath)+1:])
						self.mutex.Unlock()
					}
				}
			case err := <-watcher.Error:
				fmt.Println("error:", err)
			}
		}
	}()

	err = filepath.Walk(staticPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return watcher.Watch(path)
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

func (self *StaticVerMgr) Init(staticPath string) error {
	self.Path = staticPath
	self.Caches = make(map[string]string)
	self.mutex = &sync.Mutex{}
	self.Ignores = map[string]bool{".DS_Store": true}

	self.CacheAll(staticPath)

	go self.Moniter(staticPath)

	return nil
}

func (self *StaticVerMgr) getFileVer(url string) string {
	content, err := ioutil.ReadFile(path.Join(self.Path, url))
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
