package xweb

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

type ResponseWriter struct {
	resp       http.ResponseWriter
	buffer     []byte
	StatusCode int
	header     http.Header
}

func NewResponseWriter(resp http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		resp:       resp,
		buffer:     make([]byte, 0),
		StatusCode: 0,
		header:     make(map[string][]string),
	}
}

func (r *ResponseWriter) Header() http.Header {
	return r.header
}

func (r *ResponseWriter) Write(data []byte) (int, error) {
	if r.StatusCode == 0 {
		r.StatusCode = http.StatusOK
	}
	r.buffer = append(r.buffer, data...)
	return len(data), nil
}

func (r *ResponseWriter) Written() bool {
	return r.StatusCode != 0
}

func (r *ResponseWriter) WriteHeader(code int) {
	r.StatusCode = code
}

func (r *ResponseWriter) ServeFile(req *http.Request, path string) error {
	http.ServeFile(r, req, path)
	if r.StatusCode != http.StatusOK {
		return errors.New("serve file failed")
	}
	return nil
}

func (r *ResponseWriter) ServeReader(rd io.Reader) error {
	ln, err := io.Copy(r, rd)
	if err != nil {
		return err
	}
	r.Header().Set("Content-Length", strconv.Itoa(int(ln)))
	return nil
}

func (r *ResponseWriter) ServeXml(obj interface{}) error {
	dt, err := xml.Marshal(obj)
	if err != nil {
		return err
	}
	r.Header().Set("Content-Type", "application/xml")
	_, err = r.Write(dt)
	return err
}

func (r *ResponseWriter) ServeJson(obj interface{}) error {
	dt, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	r.Header().Set("Content-Type", "application/json")
	_, err = r.Write(dt)
	return err
}

func (r *ResponseWriter) Download(fpath string) error {
	f, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer f.Close()

	fName := filepath.Base(fpath)
	r.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%v\"", fName))
	_, err = io.Copy(r, f)
	return err
}

func (r *ResponseWriter) Flush() error {
	//fmt.Println("responsewriter:", r)

	if r.StatusCode == 0 {
		r.StatusCode = http.StatusOK
	}
	r.resp.WriteHeader(r.StatusCode)
	for key, value := range r.header {
		//fmt.Println("=====", key, value)
		if len(value) == 1 {
			r.resp.Header().Set(key, value[0])
		} else {
			for _, v := range value {
				r.resp.Header().Add(key, v)
			}
		}
	}

	_, err := r.resp.Write(r.buffer)
	if err != nil {
		return err
	}

	if flusher, ok := r.resp.(http.Flusher); ok {
		//fmt.Println("flush------>")
		flusher.Flush()
	}
	return nil
}
