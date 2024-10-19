package helper

import (
	"fmt"
	"skripsi/constant"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

type LoggerHelper interface {
	SetDebugPrefix()
	SetInfoPrefix()
	SetWarningPrefix()
	SetErrorPrefix()
	OpenOutputFile()
	CloseOutputFile()
	LogAndContinue(message string, vars ...interface{})
	LogAndExit(code int, message string, vars ...interface{})
	LogErrAndContinue(err error, message string, vars ...interface{})
	LogErrAndExit(code int, err error, message string, vars ...interface{})
}

type LoggerHelperImpl struct {
	logger *log.Logger
	file   *os.File
	prefix rune
}

func NewLoggerHelper() LoggerHelper {
	return &LoggerHelperImpl{
		logger: log.New(os.Stdout, "", log.Ldate|log.Ltime),
		prefix: 'x',
	}
}

func (h *LoggerHelperImpl) SetDebugPrefix() {
	h.logger.SetPrefix(constant.LoggerPrefixDebug)
	h.prefix = 'd'

}

func (h *LoggerHelperImpl) SetInfoPrefix() {
	h.logger.SetPrefix(constant.LoggerPrefixInfo)
	h.prefix = 'i'
}

func (h *LoggerHelperImpl) SetWarningPrefix() {
	h.logger.SetPrefix(constant.LoggerPrefixWarning)
	h.prefix = 'w'
}

func (h *LoggerHelperImpl) SetErrorPrefix() {
	h.logger.SetPrefix(constant.LoggerPrefixError)
	h.prefix = 'e'
}

func (h *LoggerHelperImpl) OpenOutputFile() {
	wd, err := os.Getwd()
	if err != nil {
		h.logger.Fatalf("Failed to get Work Directory, err: %v", err)
	}
	if h.file != nil {
		h.file.Close()
	}

	var path string
	switch h.prefix {
	case 'd':
		path = filepath.Join(wd, constant.LoggerFileDebug)
	case 'i':
		path = filepath.Join(wd, constant.LoggerFileInfo)
	case 'w':
		path = filepath.Join(wd, constant.LoggerFileWarning)
	case 'e':
		path = filepath.Join(wd, constant.LoggerFileError)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		h.logger.Fatalf("Failed to open log file, err: %v", err)
	}

	h.logger.SetOutput(file)
}

func (h *LoggerHelperImpl) CloseOutputFile() {
	if h.file != nil {
		h.file.Close()
	}
}

func (h *LoggerHelperImpl) LogAndContinue(message string, vars ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	h.logger.Printf("%s:%d \nMessage: %s", file, line, fmt.Sprintf(message, vars...))
}

func (h *LoggerHelperImpl) LogAndExit(code int, message string, vars ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	h.logger.Printf("%s:%d \nMessage: %s", file, line, fmt.Sprintf(message, vars...))
	os.Exit(code)
}

func (h *LoggerHelperImpl) LogErrAndContinue(err error, message string, vars ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	h.logger.Printf("%s:%d \nErr: %v\nMessage: %s", file, line, err, fmt.Sprintf(message, vars...))
}

func (h *LoggerHelperImpl) LogErrAndExit(code int, err error, message string, vars ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	h.logger.Printf("%s:%d \nErr: %v\nMessage: %s", file, line, err, fmt.Sprintf(message, vars...))
	os.Exit(code)
}
