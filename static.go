package xweb

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/howeyc/fsnotify"
)

func (self *StaticVersions) Moniter(staticPath string) error {
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
				if _, ok := self.ignores[filepath.Base(ev.Name)]; ok {
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
						url := ev.Name[len(self.staticDir)+1:]
						self.CacheItem(url)
					}
				} else if ev.IsDelete() {
					if d.IsDir() {
						watcher.RemoveWatch(ev.Name)
					} else {
						pa := ev.Name[len(self.staticDir)+1:]
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

func (self *StaticVersions) Run() {
	if dirExists(self.staticDir) {
		self.CacheAll(self.staticDir)

		go self.Moniter(self.staticDir)
	} else {
		self.logger.Warn("static dir", self.staticDir, "is not exist")
	}
}

func (self *StaticVersions) getFileVer(url string) string {
	fPath := filepath.Join(self.staticDir, url)
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

func (self *StaticVersions) CacheAll(staticPath string) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	err := filepath.Walk(staticPath, func(f string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		rp := f[len(staticPath)+1:]
		if _, ok := self.ignores[filepath.Base(rp)]; !ok {
			self.caches[rp] = self.getFileVer(rp)
		}
		return nil
	})
	return err
}

func (self *StaticVersions) GetVersion(url string) string {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if ver, ok := self.caches[url]; ok {
		return ver
	}

	ver := self.getFileVer(url)
	if ver != "" {
		self.caches[url] = ver
	}
	return ver
}

func (self *StaticVersions) CacheDelete(url string) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	delete(self.caches, url)
	self.logger.Infof("static file %s is deleted.\n", url)
}

func (self *StaticVersions) CacheItem(url string) {
	ver := self.getFileVer(url)
	if ver != "" {
		self.mutex.Lock()
		defer self.mutex.Unlock()
		self.caches[url] = ver
		self.logger.Infof("static file %s is created.", url)
	}
}

func (inter *StaticVersions) staticUrl(url string) string {
	ver := inter.GetVersion(url)
	if ver == "" {
		return path.Join(inter.basePath, url)
	}
	return path.Join(inter.basePath, url+"?v="+ver)
}

type StaticVersions struct {
	basePath  string
	staticDir string
	caches    map[string]string
	mutex     sync.Mutex
	ignores   map[string]bool
	logger    Logger
}

func (inter *StaticVersions) SetRender(render *Render) {
	render.FuncMaps["StaticUrl"] = func(url string) string {
		return inter.staticUrl(url)
	}
}

func NewStaticVersions(logger Logger, staticDir, basePath string) *StaticVersions {
	staticver := &StaticVersions{
		logger:    logger,
		staticDir: staticDir,
		basePath:  basePath,
		caches:    make(map[string]string),
		ignores:   map[string]bool{".DS_Store": true},
	}
	staticver.Run()
	return staticver
}

func (itor *StaticVersions) Intercept(ctx *Context) {
	ctx.Invoke()
}

type Static struct {
	RootPath   string
	IndexFiles []string
}

func (itor *Static) serveFile(ctx *Context, path string) bool {
	fPath := filepath.Join(itor.RootPath, path)
	finfo, err := os.Stat(fPath)
	if err != nil {
		if !os.IsNotExist(err) {
			ctx.HandleResult(err)
			return true
		}
	} else if !finfo.IsDir() {
		err := ctx.ServeFile(fPath)
		if err != nil {
			ctx.HandleResult(err)
		}
		return true
	}
	return false
}

func (itor *Static) Intercept(ctx *Context) {
	if ctx.Req().Method == "GET" || ctx.Req().Method == "HEAD" {
		if itor.serveFile(ctx, ctx.Req().URL.Path) {
			return
		}
	}

	ctx.Invoke()

	// try serving index.html or index.htm
	if !ctx.Resp().Written() && (ctx.Req().Method == "GET" || ctx.Req().Method == "HEAD") {
		if len(itor.IndexFiles) > 0 {
			for _, index := range itor.IndexFiles {
				if itor.serveFile(ctx, path.Join(ctx.Req().URL.Path, index)) {
					return
				}
			}
		}
	}
}
