package xweb

import (
	"fmt"
	"html/template"
	"net/http"

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

type XsrfInterceptor struct {
}

func NewXsrfInterceptor() *XsrfInterceptor {
	return &XsrfInterceptor{}
}

func (xsrf *XsrfInterceptor) SetRender(render *Render) {
	render.FuncMaps["XsrfName"] = XsrfName
}

func (inter *XsrfInterceptor) Intercept(ctx *Context) {
	if action := ctx.Action(); action != nil && ctx.Req().Method == "POST" {
		// if action implements check xsrf option and ask not check then return
		if checker, ok := action.(XsrfOptionInterface); ok && !checker.CheckXsrf() {
			ctx.Invoke()
			return
		}

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

func (c *Action) XsrfValue() string {
	var val string = ""
	cookie, err := c.GetCookie(XSRF_TAG)
	if err != nil {
		val = uuid.NewRandom().String()
		c.SetCookie(NewCookie(XSRF_TAG, val, int64(c.App.AppConfig.SessionTimeout)))
	} else {
		val = cookie.Value
	}
	return val
}

func (c *Action) XsrfFormHtml() template.HTML {
	if c.App.AppConfig.CheckXsrf {
		return template.HTML(fmt.Sprintf(`<input type="hidden" name="%v" value="%v" />`,
			XSRF_TAG, c.XsrfValue()))
	}
	return template.HTML("")
}
