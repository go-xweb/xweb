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

func NewXsrfInterceptor(app *App) *XsrfInterceptor {
	app.FuncMaps["XsrfName"] = XsrfName
	return &XsrfInterceptor{}
}

func (inter *XsrfInterceptor) Intercept(ia *Invocation) {
	action := ia.ActionContext().Action()
	if action != nil && ia.Req().Method == "POST" {
		// if action implements check xsrf option and ask not check then return
		if checker, ok := action.(XsrfOptionInterface); ok && !checker.CheckXsrf() {
			ia.Invoke()
			return
		}

		res, err := ia.Req().Cookie(XSRF_TAG)
		formVals := ia.Req().Form[XSRF_TAG]
		var formVal string
		if len(formVals) > 0 {
			formVal = formVals[0]
		}
		if err != nil || res.Value == "" || res.Value != formVal {
			ia.Resp().WriteHeader(http.StatusInternalServerError)
			ia.Resp().Write([]byte("xsrf token error."))
			return
		}
	}

	ia.Invoke()
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
