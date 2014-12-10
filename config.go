package xweb

import "time"

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
