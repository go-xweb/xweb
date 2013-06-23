package xweb

import (
	"bytes"
	"crypto/hmac"
	//"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/astaxie/beego/session"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// A Action object or it's substruct is created for every incoming HTTP request.
// It provides information
// about the request, including the http.Request object, the GET and POST params,
// and acts as a Writer for the response.
type Action struct {
	Request *http.Request
	App     *App
	http.ResponseWriter
	C            reflect.Value
	Session      session.SessionStore
	T            *T
	f            T
	BasePath     string
	RootTemplate *template.Template
}

type Mapper struct {
}

type T map[string]interface{}

// WriteString writes string data into the response object.
func (c *Action) WriteBytes(bytes []byte) error {
	_, err := c.ResponseWriter.Write(bytes)
	if err != nil {
		c.App.Server.Logger.Println("Error during write: ", err)
	}
	return err
}

func (c *Action) Write(content string, values ...interface{}) error {
	if len(values) > 0 {
		content = fmt.Sprintf(content, values...)
	}
	//c.SetHeader("Content-Length", strconv.Itoa(len(content)))
	_, err := c.ResponseWriter.Write([]byte(content))
	if err != nil {
		c.App.Server.Logger.Println("Error during write: ", err)
	}
	return err
}

// Abort is a helper method that sends an HTTP header and an optional
// body. It is useful for returning 4xx or 5xx errors.
// Once it has been called, any return value from the handler will
// not be written to the response.
func (c *Action) Abort(status int, body string) {
	c.ResponseWriter.WriteHeader(status)
	c.ResponseWriter.Write([]byte(body))
}

// Redirect is a helper method for 3xx redirects.
func (c *Action) Redirect(url string, status ...int) {
	s := 302
	if len(status) > 0 {
		s = status[0]
	}
	c.ResponseWriter.Header().Set("Location", url)
	c.ResponseWriter.WriteHeader(s)
	c.ResponseWriter.Write([]byte("Redirecting to: " + url))
}

// Notmodified writes a 304 HTTP response
func (c *Action) NotModified() {
	c.ResponseWriter.WriteHeader(304)
}

// NotFound writes a 404 HTTP response
func (c *Action) NotFound(message string) {
	c.ResponseWriter.WriteHeader(404)
	c.ResponseWriter.Write([]byte(message))
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
	c.AddHeader("Set-Cookie", cookie.String())
}

func (c *Action) GetCookie(cookieName string) (*http.Cookie, error) {
	return c.Request.Cookie(cookieName)
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
	if len(c.App.Config.CookieSecret) == 0 {
		c.App.Server.Logger.Println("Secret Key for secure cookies has not been set. Please assign a cookie secret to web.Config.CookieSecret.")
		return
	}
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	encoder.Write([]byte(val))
	encoder.Close()
	vs := buf.String()
	vb := buf.Bytes()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sig := getCookieSig(c.App.Config.CookieSecret, vb, timestamp)
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

		if getCookieSig(c.App.Config.CookieSecret, []byte(val), timestamp) != sig {
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

func (c *Action) Method() string {
	return c.Request.Method
}

func (c *Action) Go(m string, anotherc ...interface{}) {
	var t reflect.Type
	if len(anotherc) > 0 {
		t = reflect.TypeOf(anotherc[0]).Elem()
	} else {
		t = reflect.TypeOf(c.C.Interface()).Elem()
	}

	root, ok := c.App.Actions[t]
	if !ok {
		c.NotFound("Not Found")
		return
	}

	tag, ok := t.FieldByName(m)
	if !ok {
		c.NotFound("Not Found")
		return
	}

	tagStr := tag.Tag.Get("xweb")
	if tagStr != "" {
		p := tagStr
		ts := strings.Split(tagStr, " ")
		if len(ts) >= 2 {
			p = ts[1]
		}
		c.Redirect(path.Join(root, p))
	} else {
		c.Redirect(path.Join(root, m))
	}
}

func (c *Action) Flush() {
	flusher, _ := c.ResponseWriter.(http.Flusher)
	flusher.Flush()
}

func (c *Action) Include(tmplName string) interface{} {
	t := c.RootTemplate.New(tmplName)
	t.Funcs(c.getFuncs())
	content, err := ioutil.ReadFile(path.Join(c.App.Config.TemplateDir, tmplName))
	if err != nil {
		fmt.Printf("RenderTemplate %v read err\n", tmplName)
		return ""
	}
	tmpl, err := t.Parse(string(content))
	if err != nil {
		fmt.Printf("Parse %v err: %v\n", tmplName, err)
		return ""
	}
	newbytes := bytes.NewBufferString("")
	err = tmpl.Execute(newbytes, c.C.Elem().Interface())
	if err == nil {
		tplcontent, err := ioutil.ReadAll(newbytes)
		if err != nil {
			fmt.Printf("Parse %v err: %v\n", tmplName, err)
			return ""
		} else {
			return template.HTML(string(tplcontent))
		}
	} else {
		fmt.Printf("Parse %v err: %v\n", tmplName, err)
		return ""
	}
}

func (c *Action) Render(tmpl string, params ...*T) error {
	path := c.App.getTemplatePath(tmpl)
	if path != "" {
		if len(params) > 0 {
			c.T = params[0]
		}

		c.f = T{}
		c.f["include"] = c.Include

		c.RootTemplate = template.New(tmpl)
		if len(params) >= 2 {
			for k, v := range *params[1] {
				c.f[k] = v
			}
		}
		c.RootTemplate.Funcs(c.getFuncs())

		content, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Println("RenderTemplate Parse err")
			return err
		}
		tmpl, err := c.RootTemplate.Parse(string(content))
		if err == nil {
			newbytes := bytes.NewBufferString("")
			err = tmpl.Execute(newbytes, c.C.Elem().Interface())
			if err == nil {
				tplcontent, err := ioutil.ReadAll(newbytes)
				if err == nil {
					_, err = c.ResponseWriter.Write(tplcontent)
				}
			}
		}
		return err
		//}
	} else {
		return errors.New(fmt.Sprintf("No template file %v found", path))
	}
}

func (c *Action) getFuncs() template.FuncMap {
	tp := c.C.Type().Elem()
	funcs := c.App.FuncMaps[tp]
	if c.f != nil {
		for k, v := range c.f {
			funcs[k] = v
		}
	}

	return funcs
}

/*func (c *Action) RenderString(content string, params ...*T) error {
	h := md5.New()
	h.Write([]byte(content))
	name := h.Sum(nil)
	return c.NamedRender(string(name), content, params...)
}*/

func (c *Action) RenderString(content string, params ...*T) error {
	//t := c.App.RootTemplate.New("test")
	t := template.New("test")
	tp := c.C.Type().Elem()
	funcs := c.App.FuncMaps[tp]
	for k, v := range c.f {
		funcs[k] = v
	}
	if len(params) >= 2 {
		for k, v := range *params[1] {
			funcs[k] = v
		}
	}
	t.Funcs(funcs)

	tmpl, err := t.Parse(content)
	if err != nil {
		return err
	}

	if len(params) > 0 {
		c.T = params[0]
	}

	//return tmpl.Execute(c.ResponseWriter, c.C.Elem().Interface())

	newbytes := bytes.NewBufferString("")
	err = tmpl.Execute(newbytes, c.C.Elem().Interface())
	if err == nil {
		tplcontent, err := ioutil.ReadAll(newbytes)
		if err == nil {
			_, err = c.ResponseWriter.Write(tplcontent)
		}
	}

	return err
}

// SetHeader sets a response header. the current value
// of that header will be overwritten .
func (c *Action) SetHeader(key string, value string) {
	c.Header().Set(key, value)
}

// AddHeader sets a response header. it will be appended.
func (c *Action) AddHeader(key string, value string) {
	c.Header().Add(key, value)
}

func (c *Action) AddFunc(name string, tfunc interface{}) {
	if c.f == nil {
		c.f = make(map[string]interface{})
	}
	c.f[name] = tfunc
}

func (c *Action) ServeJson(obj interface{}) {
	content, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		http.Error(c.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	c.SetHeader("Content-Length", strconv.Itoa(len(content)))
	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.ResponseWriter.Write(content)
}

func (c *Action) ServeXml(obj interface{}) {
	content, err := xml.Marshal(obj)
	if err != nil {
		http.Error(c.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	c.SetHeader("Content-Length", strconv.Itoa(len(content)))
	c.ResponseWriter.Header().Set("Content-Type", "application/xml")
	c.ResponseWriter.Write(content)
}

func (c *Action) GetSlice(key string) []string {
	return c.Request.Form[key]
}

func (c *Action) GetString(key string) string {
	s := c.GetSlice(key)
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

func (c *Action) GetInt(key string) (int64, error) {
	return strconv.ParseInt(c.GetString(key), 10, 64)
}

func (c *Action) GetBool(key string) (bool, error) {
	return strconv.ParseBool(c.GetString(key))
}

func (c *Action) GetFloat(key string) (float64, error) {
	return strconv.ParseFloat(c.GetString(key), 64)
}

func (c *Action) GetFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return c.Request.FormFile(key)
}

func (c *Action) GetLogger() *log.Logger {
	return c.App.Server.Logger
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
	io.Copy(f, file)
	return nil
}

func (c *Action) StartSession() session.SessionStore {
	if c.Session == nil {
		c.Session = c.App.SessionManager.SessionStart(c.ResponseWriter, c.Request)
	}
	return c.Session
}

func (c *Action) SetSession(name interface{}, value interface{}) {
	if c.Session == nil {
		c.StartSession()
	}
	c.Session.Set(name, value)
}

func (c *Action) GetSession(name interface{}) interface{} {
	if c.Session == nil {
		c.StartSession()
	}
	return c.Session.Get(name)
}

func (c *Action) DelSession(name interface{}) {
	if c.Session == nil {
		c.StartSession()
	}
	c.Session.Delete(name)
}
