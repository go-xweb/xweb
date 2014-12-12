package xweb

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"strings"
)

type Compress struct {
	staticExts []string
}

func NewCompress(staticExts []string) *Compress {
	return &Compress{
		staticExts: staticExts,
	}
}

func (inter *Compress) Intercept(ctx *Context) {
	ctx.Invoke()

	// for cache server
	ctx.Resp().Header().Add("Vary", "Accept-Encoding")

	isStaticFileToCompress := false
	if inter.staticExts != nil && len(inter.staticExts) > 0 {
		for _, ext := range inter.staticExts {
			if strings.HasSuffix(strings.ToLower(ctx.Req().URL.Path), strings.ToLower(ext)) {
				isStaticFileToCompress = true
				break
			}
		}
	}

	if !isStaticFileToCompress {
		return
	}

	ae := ctx.Req().Header.Get("Accept-Encoding")
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
			ctx.Resp().Header().Set("Content-Encoding", "gzip")
			writer, _ = gzip.NewWriterLevel(ctx.Resp(), gzip.BestSpeed)
			break
		} else if val == "deflate" {
			ctx.Resp().Header().Set("Content-Encoding", "deflate")
			writer, _ = flate.NewWriter(ctx.Resp(), flate.BestSpeed)
			break
		}
	}

	if writer == nil {
		return
	}

	var buffer = ctx.Resp().buffer
	ctx.Resp().buffer = make([]byte, 0)
	writer.Write(buffer)
	switch writer.(type) {
	case *gzip.Writer:
		writer.(*gzip.Writer).Close()
	case *flate.Writer:
		writer.(*flate.Writer).Close()
	}
}
