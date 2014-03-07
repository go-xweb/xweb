package xweb

import (
	"fmt"
	"log"
)

const (
	ForeBlack  = iota + 30 //30         40         黑色
	ForeRed                //31         41         紅色
	ForeGreen              //32         42         綠色
	ForeYellow             //33         43         黃色
	ForeBlue               //34         44         藍色
	ForePurple             //35         45         紫紅色
	ForeCyan               //36         46         青藍色
	ForeWhite              //37         47         白色
)

const (
	LevelTrace = iota + 1
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelCritical
)

type osLogger func(*log.Logger, int, string, ...interface{})

func winLog(logger *log.Logger, color int, format string, params ...interface{}) {
	logger.Printf(format, params...)
}

func unixLog(logger *log.Logger, color int, format string, params ...interface{}) {
	s := fmt.Sprintf(format, params...)
	logger.Printf("\033[%v;1m%s\033[0m", color, s)
}
