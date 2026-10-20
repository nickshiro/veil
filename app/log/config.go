package log

import (
	"strconv"

	clog "github.com/nickshiro/veil/common/log"
)

type LogType int32

const (
	LogType_None    LogType = 0
	LogType_Console LogType = 1
	LogType_File    LogType = 2
	LogType_Event   LogType = 3
)

func (t LogType) String() string {
	switch t {
	case LogType_None:
		return "None"
	case LogType_Console:
		return "Console"
	case LogType_File:
		return "File"
	case LogType_Event:
		return "Event"
	default:
		return strconv.FormatInt(int64(t), 10)
	}
}

type Config struct {
	ErrorLogType  LogType
	ErrorLogLevel clog.Severity
	ErrorLogPath  string
	AccessLogType LogType
	AccessLogPath string
	EnableDnsLog  bool
	MaskAddress   string
}
