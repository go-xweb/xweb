package xweb

import (
	"sync"
	"time"

	"github.com/lunny/tango"
)

type AppConfig struct {
	Mode              int
	StaticDir         string
	TemplateDir       string
	SessionOn         bool
	SessionTimeout    time.Duration
	MaxUploadSize     int64
	CookieSecret      string
	StaticFileVersion bool
	CacheTemplates    bool
	ReloadTemplates   bool
	CheckXsrf         bool
	FormMapToStruct   bool
	EnableHttpCache   bool
}

var DefaultAppConfig = AppConfig{
	Mode:              Product,
	StaticDir:         "static",
	TemplateDir:       "templates",
	SessionOn:         true,
	SessionTimeout:    3600,
	MaxUploadSize:     10 * 1024 * 1024,
	StaticFileVersion: true,
	CacheTemplates:    true,
	ReloadTemplates:   true,
	CheckXsrf:         true,
	FormMapToStruct:   true,
}

type cfgs map[string]interface{}
type Configs struct {
	cfgs
	lock sync.RWMutex
}

func NewConfigs() *Configs {
	return &Configs{
		cfgs: make(map[string]interface{}),
	}
}

func (cfgs *Configs) SetConfig(name string, val interface{}) {
	cfgs.lock.Lock()
	defer cfgs.lock.Unlock()
	cfgs.cfgs[name] = val
}

func (cfgs *Configs) GetConfig(name string) interface{} {
	cfgs.lock.RLock()
	defer cfgs.lock.RUnlock()
	return cfgs.cfgs[name]
}

type ConfigsInterface interface {
	SetConfigs(configs *Configs)
}

func (cfgs *Configs) Handle(ctx *tango.Context) {
	if action := ctx.Action(); action != nil {
		if c, ok := action.(ConfigsInterface); ok {
			c.SetConfigs(cfgs)
		}
	}

	ctx.Next()
}
