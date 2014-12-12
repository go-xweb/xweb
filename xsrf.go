package xweb

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/go-xweb/uuid"
)

const (
	XSRF_TAG string = "_xsrf"
)

func XsrfName() string {
	return XSRF_TAG
}

type XsrfOptionInterface interface {
	CheckXsrf() bool
}

type Xsrf struct {
	render *Render
	timeout time.Duration
}

func NewXsrf(timeout time.Duration) *Xsrf {
	return &Xsrf{
		timeout:timeout,
	}
}

func (xsrf *Xsrf) SetRender(render *Render) {
	xsrf.render = render
	render.FuncMaps["XsrfName"] = XsrfName
}

func (xsrf *Xsrf) Intercept(ctx *Context) {
	var action interface{}
	if action = ctx.Action(); action == nil {
		ctx.Invoke()
		return
	}

	// if action implements check xsrf option and ask not check then return
	if checker, ok := action.(XsrfOptionInterface); ok && !checker.CheckXsrf() {
		ctx.Invoke()
		return
	}

	if ctx.Req().Method == "GET" {
		xsrf.render.FuncMaps["XsrfName"] = XSRF_TAG

		var val string = ""
		cookie, err := ctx.Req().Cookie(XSRF_TAG)
		if err != nil {
			val = uuid.NewRandom().String()
			cookie = NewCookie(XSRF_TAG, val, int64(xsrf.timeout))
			ctx.Resp().SetHeader("Set-Cookie", cookie.String())
		} else {
			val = cookie.Value
		}

		xsrf.render.FuncMaps["XsrfValue"] = func() string {
			return val
		}
		xsrf.render.FuncMaps["XsrfFormHtml"] = func() template.HTML {
			return template.HTML(fmt.Sprintf(`<input type="hidden" name="%v" value="%v" />`,
			XSRF_TAG, val))
		}
	} else if ctx.Req().Method == "POST" {
		res, err := ctx.Req().Cookie(XSRF_TAG)
		formVals := ctx.Req().Form[XSRF_TAG]
		var formVal string
		if len(formVals) > 0 {
			formVal = formVals[0]
		}
		if err != nil || res.Value == "" || res.Value != formVal {
			ctx.Resp().WriteHeader(http.StatusInternalServerError)
			ctx.Resp().Write([]byte("xsrf token error."))
			return
		}
	}

	ctx.Invoke()
}