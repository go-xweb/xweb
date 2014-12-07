package xweb

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"strconv"
)

type ResponseWriter struct {
	resp       http.ResponseWriter
	StatusCode int
}

func NewResponseWriter(resp http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{resp, 0}
}

func (r *ResponseWriter) Header() http.Header {
	return r.resp.Header()
}

func (r *ResponseWriter) Write(data []byte) (int, error) {
	return r.resp.Write(data)
}

func (r *ResponseWriter) Written() bool {
	return r.StatusCode != 0
}

func (r *ResponseWriter) WriteHeader(code int) {
	r.StatusCode = code
	r.resp.WriteHeader(code)
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
	r.resp.Header().Set("Content-Length", strconv.Itoa(int(ln)))
	return nil
}

func (r *ResponseWriter) ServeXml(obj interface{}) error {
	dt, err := xml.Marshal(obj)
	if err != nil {
		return err
	}
	r.resp.Header().Set("Content-Type", "application/xml")
	_, err = r.Write(dt)
	return err
}

func (r *ResponseWriter) ServeJson(obj interface{}) error {
	dt, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	r.resp.Header().Set("Content-Type", "application/json")
	_, err = r.Write(dt)
	return err
}

func (r *ResponseWriter) Flush() {
	flusher, _ := r.resp.(http.Flusher)
	flusher.Flush()
}
