package xweb

import (
	"net/http"
)

type XsrfInterceptor struct {
}

func (inter *XsrfInterceptor) Intercept(ia *Invocation) {
	if ia.Req().Method == "POST" {
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
