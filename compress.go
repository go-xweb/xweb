package xweb

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"strings"
)

type GZipInterceptor struct {
}

func (inter *GZipInterceptor) Intercept(ia *Invocation) {
	ia.Invoke()

	if ia.Req().Header.Get("Accept-Encoding") != "" {
		splitted := strings.SplitN(ia.Req().Header.Get("Accept-Encoding"), ",", -1)
		encodings := make([]string, len(splitted))
		for i, val := range splitted {
			encodings[i] = strings.TrimSpace(val)
		}
		var writer io.Writer
		for _, val := range encodings {
			if val == "gzip" {
				ia.Resp().Header().Set("Content-Encoding", "gzip")
				writer, _ = gzip.NewWriterLevel(ia.Resp(), gzip.BestSpeed)
				break
			} else if val == "deflate" {
				ia.Resp().Header().Set("Content-Encoding", "deflate")
				writer, _ = flate.NewWriter(ia.Resp(), flate.BestSpeed)
				break
			}
		}

		if writer == nil {
			return
		}

		var buffer = ia.Resp().buffer
		ia.Resp().buffer = make([]byte, 0)
		writer.Write(buffer)
		switch writer.(type) {
		case *gzip.Writer:
			writer.(*gzip.Writer).Close()
		case *flate.Writer:
			writer.(*flate.Writer).Close()
		}
	}
}
