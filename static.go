package xweb

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/go-xweb/log"
	"github.com/howeyc/fsnotify"
)

type StaticVerMgr struct {
	Caches  map[string]string
	mutex   *sync.Mutex
	Path    string
	Ignores map[string]bool
	logger  *log.Logger
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
						self.CacheItem(url)
					}
				} else if ev.IsDelete() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						pa := ev.Name[len(self.Path)+1:]
						self.CacheDelete(pa)
					}
				} else if ev.IsModify() {
					if d.IsDir() {
					} else {
						url := ev.Name[len(staticPath)+1:]
						self.CacheItem(url)
					}
				} else if ev.IsRename() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						url := ev.Name[len(staticPath)+1:]
						self.CacheDelete(url)
					}
				}
			case err := <-watcher.Error:
				self.logger.Errorf("error: %v", err)
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

func (self *StaticVerMgr) Init(staticPath string) error {
	self.Path = staticPath
	self.Caches = make(map[string]string)
	self.mutex = &sync.Mutex{}
	self.Ignores = map[string]bool{".DS_Store": true}

	if dirExists(staticPath) {
		self.CacheAll(staticPath)

		go self.Moniter(staticPath)
	}

	return nil
}

func (self *StaticVerMgr) getFileVer(url string) string {
	fPath := filepath.Join(self.Path, url)
	self.logger.Debug("loaded static ", fPath)
	f, err := os.Open(fPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	fInfo, err := f.Stat()
	if err != nil {
		return ""
	}

	var maxRead = fInfo.Size()
	if maxRead > 1024*1024*20 {
		maxRead = 1024 * 1024 * 20
	}

	content := make([]byte, int(maxRead))
	_, err = f.Read(content)
	if err == nil {
		h := md5.New()
		io.WriteString(h, string(content))
		io.WriteString(h, fmt.Sprintf("%d", fInfo.Size()))
		return fmt.Sprintf("%x", h.Sum(nil))[0:4]
	}
	return ""
}

func (self *StaticVerMgr) CacheAll(staticPath string) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	//fmt.Print("Getting static file version number, please wait... ")
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
	//fmt.Println("Complete.")
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

func (self *StaticVerMgr) CacheDelete(url string) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	delete(self.Caches, url)
	self.logger.Infof("static file %s is deleted.\n", url)
}

func (self *StaticVerMgr) CacheItem(url string) {
	ver := self.getFileVer(url)
	if ver != "" {
		self.mutex.Lock()
		defer self.mutex.Unlock()
		self.Caches[url] = ver
		self.logger.Infof("static file %s is created.", url)
	}
}

func (a *App) StaticUrlNoVer(url string) string {
	var basePath string
	if a.AppConfig.StaticDir == RootApp().AppConfig.StaticDir {
		basePath = RootApp().BasePath
	} else {
		basePath = a.BasePath
	}

	return path.Join(basePath, url)
}

func (a *App) StaticUrl(url string, getver func(string) string) string {
	var basePath string
	if a.AppConfig.StaticDir == RootApp().AppConfig.StaticDir {
		basePath = RootApp().BasePath
	} else {
		basePath = a.BasePath
	}

	ver := getver(url)
	if ver == "" {
		return path.Join(basePath, url)
	}
	return path.Join(basePath, url+"?v="+ver)
}

type StaticVerInterceptor struct {
	staticMgr *StaticVerMgr
}

func NewStaticVerInterceptor(logger *log.Logger, staticDir string, app *App) *StaticVerInterceptor {
	staticMgr := &StaticVerMgr{
		logger: logger,
	}
	staticMgr.Init(staticDir)

	// TODO: refactoring this
	app.FuncMaps["StaticUrl"] = func(url string) string {
		return app.StaticUrl(url, staticMgr.GetVersion)
	}

	return &StaticVerInterceptor{
		staticMgr: staticMgr,
	}
}

func (itor *StaticVerInterceptor) Intercept(ia *Invocation) {
	ia.Invoke()
}

type StaticInterceptor struct {
	RootPath   string
	IndexFiles []string
}

func (itor *StaticInterceptor) serveFile(ai *Invocation, path string) bool {
	fPath := filepath.Join(itor.RootPath, path)
	finfo, err := os.Stat(fPath)
	if err != nil {
		if !os.IsNotExist(err) {
			ai.HandleResult(err)
			return true
		}
	} else if !finfo.IsDir() {
		err := ai.ServeFile(fPath)
		if err != nil {
			ai.HandleResult(err)
		}
		return true
	}
	return false
}

func (itor *StaticInterceptor) Intercept(ai *Invocation) {
	if ai.Req().Method == "GET" || ai.Req().Method == "HEAD" {
		if itor.serveFile(ai, ai.Req().URL.Path) {
			return
		}
	}

	ai.Invoke()

	// try serving index.html or index.htm
	if !ai.Resp().Written() && (ai.Req().Method == "GET" || ai.Req().Method == "HEAD") {
		if len(itor.IndexFiles) > 0 {
			for _, index := range itor.IndexFiles {
				if itor.serveFile(ai, path.Join(ai.Req().URL.Path, index)) {
					return
				}
			}
		}
	}
}
