package xweb

import (
	"github.com/lunny/tango"
)

func NotSupported(content ...string) error {
	return tango.NotSupported(content...)
}
