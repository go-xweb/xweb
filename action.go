package xweb

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-xweb/httpsession"
)

// An Action object or it's substruct is created for every incoming HTTP request.
// It provides information
// about the request, including the http.Request object, the GET and POST params,
// and acts as a Writer for the response.
type Action struct {
	App *App
	*Configs
	Logger

	*Request
	*ResponseWriter
	session *httpsession.Session
	*Renderer

	C reflect.Value
}

func (c *Action) DisableHttpCache() {
	c.SetHeader("Expires", "Mon, 26 Jul 1997 05:00:00 GMT")
	c.SetHeader("Last-Modified", webTime(time.Now().UTC()))
	c.SetHeader("Cache-Control", "no-store, no-cache, must-revalidate")
	c.SetHeader("Cache-Control", "post-check=0, pre-check=0")
	c.SetHeader("Pragma", "no-cache")
}

func (c *Action) HttpCache(content []byte) bool {
	h := md5.New()
	h.Write(content)
	Etag := hex.EncodeToString(h.Sum(nil))
	//c.SetHeader("Connection", "keep-alive")
	c.SetHeader("X-Cache", "HIT from COSCMS-Page-Cache")
	//c.SetHeader("X-Cache", "HIT from COSCMS-Page-Cache 2013-12-02 17:16:01")
	if inm := c.Request.Header("If-None-Match"); inm != "" && inm == Etag {
		h := c.ResponseWriter.Header()
		delete(h, "Content-Type")
		delete(h, "Content-Length")
		c.ResponseWriter.WriteHeader(http.StatusNotModified)
		return true
	}
	c.SetHeader("Etag", Etag)
	c.SetHeader("Cache-Control", "public,max-age=1")
	return false
}

// Body sets response body content.
// if EnableGzip, compress content string.
// it sends out response body directly.
func (c *Action) SetBody(content []byte) error {
	if c.App.AppConfig.EnableHttpCache && c.HttpCache(content) {
		return nil
	}

	c.SetHeader("Content-Length", strconv.Itoa(len(content)))
	_, err := c.ResponseWriter.Write(content)
	return err
}

// WriteString writes string data into the response object.
func (c *Action) WriteBytes(bytes []byte) error {
	return c.SetBody(bytes)
}

func (c *Action) Write(content string, values ...interface{}) error {
	if len(values) > 0 {
		content = fmt.Sprintf(content, values...)
	}
	return c.SetBody([]byte(content))
}

// +inject
func (c *Action) SetRenderer(renderer *Renderer) {
	c.Renderer = renderer
}

// +inject
// func (c *Action) SetRequest(req *http.Request) {
func (c *Action) SetRequest(req *Request) {
	c.Request = req
}

// +inject
func (c *Action) SetResponse(resp *ResponseWriter) {
	c.ResponseWriter = resp
}

// +inject
func (c *Action) SetApp(app *App) {
	c.App = app
}

// +inject
func (c *Action) SetLogger(logger Logger) {
	c.Logger = logger
}

// +inject
func (c *Action) SetSessions(session *httpsession.Session) {
	c.session = session
}

// +inject
func (c *Action) SetConfigs(configs *Configs) {
	c.Configs = configs
}

// ContentType sets the Content-Type header for an HTTP response.
// For example, c.ContentType("json") sets the content-type to "application/json"
// If the supplied value contains a slash (/) it is set as the Content-Type
// verbatim. The return value is the content type as it was
// set, or an empty string if none was found.
func (c *Action) SetContentType(val string) string {
	var ctype string
	if strings.ContainsRune(val, '/') {
		ctype = val
	} else {
		if !strings.HasPrefix(val, ".") {
			val = "." + val
		}
		ctype = mime.TypeByExtension(val)
	}
	if ctype != "" {
		c.SetHeader("Content-Type", ctype)
	}
	return ctype
}

// SetCookie adds a cookie header to the response.
func (c *Action) SetCookie(cookie *http.Cookie) {
	c.SetHeader("Set-Cookie", cookie.String())
}

func getCookieSig(key string, val []byte, timestamp string) string {
	hm := hmac.New(sha1.New, []byte(key))

	hm.Write(val)
	hm.Write([]byte(timestamp))

	hex := fmt.Sprintf("%02x", hm.Sum(nil))
	return hex
}

func (c *Action) SetSecureCookie(name string, val string, age int64) {
	//base64 encode the val
	if len(c.App.AppConfig.CookieSecret) == 0 {
		c.Logger.Error("Secret Key for secure cookies has not been set. Please assign a cookie secret to web.Config.CookieSecret.")
		return
	}
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	encoder.Write([]byte(val))
	encoder.Close()
	vs := buf.String()
	vb := buf.Bytes()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sig := getCookieSig(c.App.AppConfig.CookieSecret, vb, timestamp)
	cookie := strings.Join([]string{vs, timestamp, sig}, "|")
	c.SetCookie(NewCookie(name, cookie, age))
}

func (c *Action) GetSecureCookie(name string) (string, bool) {
	for _, cookie := range c.Request.Cookies() {
		if cookie.Name != name {
			continue
		}

		parts := strings.SplitN(cookie.Value, "|", 3)

		val := parts[0]
		timestamp := parts[1]
		sig := parts[2]

		if getCookieSig(c.App.AppConfig.CookieSecret, []byte(val), timestamp) != sig {
			return "", false
		}

		ts, _ := strconv.ParseInt(timestamp, 0, 64)

		if time.Now().Unix()-31*86400 > ts {
			return "", false
		}

		buf := bytes.NewBufferString(val)
		encoder := base64.NewDecoder(base64.StdEncoding, buf)

		res, _ := ioutil.ReadAll(encoder)
		return string(res), true
	}
	return "", false
}

func (c *Action) Go(m string, anotherc ...interface{}) error {
	var t reflect.Type
	if len(anotherc) > 0 {
		t = reflect.TypeOf(anotherc[0]).Elem()
	} else {
		t = c.C.Type().Elem()
	}

	root, ok := c.App.ActionsPath[t]
	if !ok {
		return NotFound()
	}

	uris := strings.Split(m, "?")

	tag, ok := t.FieldByName(uris[0])
	if !ok {
		return NotFound()
	}

	tagStr := tag.Tag.Get("xweb")
	var rPath string
	if tagStr != "" {
		p := tagStr
		ts := strings.Split(tagStr, " ")
		if len(ts) >= 2 {
			p = ts[1]
		}
		rPath = path.Join(root, p, m[len(uris[0]):])
	} else {
		rPath = path.Join(root, m)
	}
	rPath = strings.Replace(rPath, "//", "/", -1)
	return c.Redirect(rPath)
}

func (c *Action) BasePath() string {
	return c.App.BasePath
}

func (c *Action) Namespace() string {
	return c.App.ActionsPath[c.C.Type()]
}

func (c *Action) SetConfig(name string, value interface{}) {
	c.App.Config[name] = value
}

func (c *Action) GetConfig(name string) interface{} {
	return c.App.Config[name]
}

func (c *Action) SaveToFile(fromfile, tofile string) error {
	file, _, err := c.Request.FormFile(fromfile)
	if err != nil {
		return err
	}
	defer file.Close()
	f, err := os.OpenFile(tofile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, file)
	return err
}

func (c *Action) Session() *httpsession.Session {
	return c.session
}

func (c *Action) GetSession(key string) interface{} {
	return c.Session().Get(key)
}

func (c *Action) SetSession(key string, value interface{}) {
	c.Session().Set(key, value)
}

func (c *Action) DelSession(key string) {
	c.Session().Del(key)
}
