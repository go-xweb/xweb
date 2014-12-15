package xweb

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lunny/tango"
)

// a struct implements this interface can be convert from request param to a struct
type FromConversion interface {
	FromString(content string) error
}

// a struct implements this interface can be convert from struct to template variable
// Not Implemented
type ToConversion interface {
	ToString() string
}

func namedStructMap(logger tango.Logger, vc reflect.Value, r *http.Request, topName string) error {
	r.ParseForm()
	for k, t := range r.Form {
		if topName != "" {
			if !strings.HasPrefix(k, topName) {
				continue
			}
			k = k[len(topName)+1:]
		}

		v := t[0]
		names := strings.Split(k, ".")
		var err error
		if len(names) == 1 {
			names, err = splitJson(k)
			if err != nil {
				logger.Warn("Unrecognize form key", k, err)
				continue
			}
		}

		var value reflect.Value = vc
		for i, name := range names {
			name = strings.Title(name)
			if i != len(names)-1 {
				if value.Kind() != reflect.Struct {
					logger.Warnf("arg error, value kind is %v", value.Kind())
					break
				}

				value = value.FieldByName(name)
				if !value.IsValid() {
					logger.Warnf("(%v value is not valid %v)", name, value)
					break
				}
				if !value.CanSet() {
					logger.Warnf("can not set %v -> %v", name, value.Interface())
					break
				}

				if value.Kind() == reflect.Ptr {
					if value.IsNil() {
						value.Set(reflect.New(value.Type().Elem()))
					}
					value = value.Elem()
				}
			} else {
				if value.Kind() != reflect.Struct {
					logger.Warnf("arg error, value %v kind is %v", name, value.Kind())
					break
				}
				tv := value.FieldByName(name)
				if !tv.IsValid() {
					break
				}
				if !tv.CanSet() {
					logger.Warnf("can not set %v to %v", k, tv)
					break
				}

				if tv.Kind() == reflect.Ptr {
					tv.Set(reflect.New(tv.Type().Elem()))
					tv = tv.Elem()
				}

				var l interface{}
				switch k := tv.Kind(); k {
				case reflect.String:
					l = v
					tv.Set(reflect.ValueOf(l))
				case reflect.Bool:
					l = (v != "false" && v != "0")
					tv.Set(reflect.ValueOf(l))
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
					x, err := strconv.Atoi(v)
					if err != nil {
						logger.Warnf("arg %v as int: %v", v, err)
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Int64:
					x, err := strconv.ParseInt(v, 10, 64)
					if err != nil {
						logger.Warnf("arg %v as int64: %v", v, err)
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Float32, reflect.Float64:
					x, err := strconv.ParseFloat(v, 64)
					if err != nil {
						logger.Warnf("arg %v as float64: %v", v, err)
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					x, err := strconv.ParseUint(v, 10, 64)
					if err != nil {
						logger.Warnf("arg %v as uint: %v", v, err)
						break
					}
					l = x
					tv.Set(reflect.ValueOf(l))
				case reflect.Struct:
					if tvf, ok := tv.Interface().(FromConversion); ok {
						err := tvf.FromString(v)
						if err != nil {
							logger.Warnf("struct %v invoke FromString faild", tvf)
						}
					} else if tv.Type().String() == "time.Time" {
						x, err := time.Parse("2006-01-02 15:04:05.000 -0700", v)
						if err != nil {
							x, err = time.Parse("2006-01-02 15:04:05", v)
							if err != nil {
								x, err = time.Parse("2006-01-02", v)
								if err != nil {
									logger.Warnf("unsupported time format %v, %v", v, err)
									break
								}
							}
						}
						l = x
						tv.Set(reflect.ValueOf(l))
					} else {
						logger.Warn("can not set an struct which is not implement Fromconversion interface")
					}
				case reflect.Ptr:
					logger.Warn("can not set an ptr of ptr")
				case reflect.Slice, reflect.Array:
					tt := tv.Type().Elem()
					tk := tt.Kind()
					if tk == reflect.String {
						tv.Set(reflect.ValueOf(t))
						break
					}

					if tv.IsNil() {
						tv.Set(reflect.MakeSlice(tv.Type(), len(t), len(t)))
					}

					for i, s := range t {
						var err error
						switch tk {
						case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int8, reflect.Int64:
							var v int64
							v, err = strconv.ParseInt(s, 10, tt.Bits())
							if err == nil {
								tv.Index(i).SetInt(v)
							}
						case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
							var v uint64
							v, err = strconv.ParseUint(s, 10, tt.Bits())
							if err == nil {
								tv.Index(i).SetUint(v)
							}
						case reflect.Float32, reflect.Float64:
							var v float64
							v, err = strconv.ParseFloat(s, tt.Bits())
							if err == nil {
								tv.Index(i).SetFloat(v)
							}
						case reflect.Bool:
							var v bool
							v, err = strconv.ParseBool(s)
							if err == nil {
								tv.Index(i).SetBool(v)
							}
						case reflect.Complex64, reflect.Complex128:
							// TODO:
							err = fmt.Errorf("unsupported slice element type %v", tk.String())
						default:
							err = fmt.Errorf("unsupported slice element type %v", tk.String())
						}
						if err != nil {
							logger.Warnf("slice error: %v, %v", name, err)
							break
						}
					}
				default:
					logger.Warnf("unknow mapping method", name)
					break
				}
			}
		}
	}
	return nil
}

func redirect(w http.ResponseWriter, url string, status ...int) error {
	s := 302
	if len(status) > 0 {
		s = status[0]
	}
	w.Header().Set("Location", url)
	w.WriteHeader(s)
	_, err := w.Write([]byte("Redirecting to: " + url))
	return err
}

// the func is the same as condition ? true : false
func Ternary(express bool, trueVal interface{}, falseVal interface{}) interface{} {
	if express {
		return trueVal
	}
	return falseVal
}

// internal utility methods
func webTime(t time.Time) string {
	ftime := t.Format(time.RFC1123)
	if strings.HasSuffix(ftime, "UTC") {
		ftime = ftime[0:len(ftime)-3] + "GMT"
	}
	return ftime
}

func JoinPath(paths ...string) string {
	if len(paths) < 1 {
		return ""
	}
	res := ""
	for _, p := range paths {
		res = path.Join(res, p)
	}
	return res
}

func PageSize(total, limit int) int {
	if total <= 0 {
		return 1
	} else {
		x := total % limit
		if x > 0 {
			return total/limit + 1
		} else {
			return total / limit
		}
	}
}

func SimpleParse(data string) map[string]string {
	configs := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		vs := strings.Split(line, "=")
		if len(vs) == 2 {
			configs[strings.TrimSpace(vs[0])] = strings.TrimSpace(vs[1])
		}
	}
	return configs
}

func dirExists(dir string) bool {
	d, e := os.Stat(dir)
	switch {
	case e != nil:
		return false
	case !d.IsDir():
		return false
	}

	return true
}

func fileExists(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

// Urlencode is a helper method that converts a map into URL-encoded form data.
// It is a useful when constructing HTTP POST requests.
func Urlencode(data map[string]string) string {
	var buf bytes.Buffer
	for k, v := range data {
		buf.WriteString(url.QueryEscape(k))
		buf.WriteByte('=')
		buf.WriteString(url.QueryEscape(v))
		buf.WriteByte('&')
	}
	s := buf.String()
	return s[0 : len(s)-1]
}

func UnTitle(s string) string {
	if len(s) < 2 {
		return strings.ToLower(s)
	}
	return strings.ToLower(string(s[0])) + s[1:]
}

var slugRegex = regexp.MustCompile(`(?i:[^a-z0-9\-_])`)

// Slug is a helper function that returns the URL slug for string s.
// It's used to return clean, URL-friendly strings that can be
// used in routing.
func Slug(s string, sep string) string {
	if s == "" {
		return ""
	}
	slug := slugRegex.ReplaceAllString(s, sep)
	if slug == "" {
		return ""
	}
	quoted := regexp.QuoteMeta(sep)
	sepRegex := regexp.MustCompile("(" + quoted + "){2,}")
	slug = sepRegex.ReplaceAllString(slug, sep)
	sepEnds := regexp.MustCompile("^" + quoted + "|" + quoted + "$")
	slug = sepEnds.ReplaceAllString(slug, "")
	return strings.ToLower(slug)
}

// NewCookie is a helper method that returns a new http.Cookie object.
// Duration is specified in seconds. If the duration is zero, the cookie is permanent.
// This can be used in conjunction with ctx.SetCookie.
func NewCookie(name string, value string, age int64) *http.Cookie {
	var utctime time.Time
	if age == 0 {
		// 2^31 - 1 seconds (roughly 2038)
		utctime = time.Unix(2147483647, 0)
	} else {
		utctime = time.Unix(time.Now().Unix()+age, 0)
	}
	return &http.Cookie{Name: name, Value: value, Expires: utctime}
}

func removeStick(uri string) string {
	uri = strings.TrimRight(uri, "/")
	if uri == "" {
		uri = "/"
	}
	return uri
}

var (
	fieldCache      = make(map[reflect.Type]map[string]int)
	fieldCacheMutex sync.RWMutex
)

// user[name][test]
func splitJson(s string) ([]string, error) {
	res := make([]string, 0)
	var begin, end int
	var isleft bool
	for i, r := range s {
		switch r {
		case '[':
			isleft = true
			if i > 0 && s[i-1] != ']' {
				if begin == end {
					return nil, errors.New("unknow character")
				}
				res = append(res, s[begin:end+1])
			}
			begin = i + 1
			end = begin
		case ']':
			if !isleft {
				return nil, errors.New("unknow character")
			}
			isleft = false
			if begin != end {
				//return nil, errors.New("unknow character")

				res = append(res, s[begin:end+1])
				begin = i + 1
				end = begin
			}
		default:
			end = i
		}
		if i == len(s)-1 && begin != end {
			res = append(res, s[begin:end+1])
		}
	}
	return res, nil
}

// this method cache fields' index to field name
func fieldByName(v reflect.Value, name string) reflect.Value {
	t := v.Type()
	fieldCacheMutex.RLock()
	cache, ok := fieldCache[t]
	fieldCacheMutex.RUnlock()
	if !ok {
		cache = make(map[string]int)
		for i := 0; i < v.NumField(); i++ {
			cache[t.Field(i).Name] = i
		}
		fieldCacheMutex.Lock()
		fieldCache[t] = cache
		fieldCacheMutex.Unlock()
	}

	if i, ok := cache[name]; ok {
		return v.Field(i)
	}

	return reflect.Zero(t)
}
