package xweb

import (
	//"fmt"
	"html/template"
	"time"
)

/*func Add(left interface{}, right interface{}) interface{} {
	tLeft := left.(type)
	tRight := right.(type)
	if tLeft == tRight {
		return
	}
	 {
		case int, int64:
			switch right.(type) {
			case int, int8, int16, int32, int64:
				return left.
			}
	}
}*/

func Add(left int, right int) int {
	return left + right
}

func FormatDate(t time.Time, format string) string {
	return t.Format(format)
}

func Eq(left interface{}, right interface{}) bool {
	return left == right
}

func Html(raw string) template.HTML {
	return template.HTML(raw)
}

var (
	defaultFuncs template.FuncMap = template.FuncMap{
		"Eq":         Eq,
		"FormatDate": FormatDate,
		"Html":       Html,
		"Add":        Add,
	}
)
