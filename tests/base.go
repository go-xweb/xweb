package tests

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"net/http"
)

func gzipDecode(src []byte) ([]byte, error) {
	rd := bytes.NewReader(src)
	b, err := gzip.NewReader(rd)
	if err != nil {
		return nil, err
	}

	defer b.Close()

	data, err := ioutil.ReadAll(b)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func get(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.Header.Get("Content-Encoding") == "gzip" {
		data, err := gzipDecode(bs)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return string(bs), nil
}
