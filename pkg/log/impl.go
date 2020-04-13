package log

import (
	"fmt"
	"log"
	"strings"
)

type level int

const (
	ERROR level = iota
	WARN
	INFO
	DEBUG
)

var (
	currentLevel = INFO
)

func (l level) String() string {
	s := []string{
		"ERROR",
		"WARN",
		"INFO",
		"DEBUG",
	}
	return s[l]
}

func SetLevel(l string) {
	switch strings.ToLower(l) {
	default:
		currentLevel = DEBUG
		Debug("Defaulting to debug level logging")
	case "info":
		currentLevel = INFO
	case "warn":
		currentLevel = WARN
	case "error":
		currentLevel = ERROR
	}
}

func logAtLevel(l level, args ...interface{}) {
	if l <= currentLevel {
		log.Println(l, args)
	}
}
func Infof(pattern string, args ...interface{}) {
	logAtLevel(INFO, fmt.Sprintf(pattern, args...))
}
func Info(args ...interface{}) {
	logAtLevel(INFO, args)
}
func Debug(args ...interface{}) {
	logAtLevel(DEBUG, args)
}
func Error(args ...interface{}) {
	logAtLevel(ERROR, args)
}
func Warn(args ...interface{}) {
	logAtLevel(WARN, args)
}
