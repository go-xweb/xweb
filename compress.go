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

	// for cache server
	ia.Resp().Header().Add("Vary", "Accept-Encoding")

	isStaticFileToCompress := false
	if ia.app.Server.Config.StaticExtensionsToGzip != nil && len(ia.app.Server.Config.StaticExtensionsToGzip) > 0 {
		for _, statExtension := range ia.app.Server.Config.StaticExtensionsToGzip {
			if strings.HasSuffix(strings.ToLower(ia.Req().URL.Path), strings.ToLower(statExtension)) {
				isStaticFileToCompress = true
				break
			}
		}
	}

	if !isStaticFileToCompress {
		return
	}

	ae := ia.Req().Header.Get("Accept-Encoding")
	if ae == "" {
		return
	}

	splitted := strings.SplitN(ae, ",", -1)
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
