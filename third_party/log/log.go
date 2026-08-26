package log

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var stdoutLogger = logrus.New()
var stderrLogger = logrus.New()

func init() {
	stdoutLogger.Out = os.Stdout
	stderrLogger.Out = os.Stderr
}

func SetOutput(out io.Writer) {
	stdoutLogger.Out = out
	stderrLogger.Out = out
}

// Info is wrapper for logrus.Info to print to stdout
func Info(args ...interface{}) {
	stdoutLogger.Info(safeMessage(fmt.Sprint(args...)))
}

// Debug is wrapper for logrus.Debug to print to stdout
func Debug(args ...interface{}) {
	stdoutLogger.Debug(safeMessage(fmt.Sprint(args...)))
}

// Error is wrapper for logrus.Error to print to stderr
func Error(args ...interface{}) {
	stderrLogger.Error(safeMessage(fmt.Sprint(args...)))
}

// Fatal is wrapper for logrus.Fatal to print to stderr
func Fatal(args ...interface{}) {
	stderrLogger.Fatal(safeMessage(fmt.Sprint(args...)))
}

// Warn is wrapper for logrus.Warn to print to stderr
func Warn(args ...interface{}) {
	stderrLogger.Warn(safeMessage(fmt.Sprint(args...)))
}

// Infof is wrapper for logrus.Infof to print to stdout
func Infof(format string, args ...interface{}) {
	stdoutLogger.Info(safeMessage(fmt.Sprintf(format, args...)))
}

// Debugf is wrapper for logrus.Debugf to print to stdout
func Debugf(format string, args ...interface{}) {
	stdoutLogger.Debug(safeMessage(fmt.Sprintf(format, args...)))
}

// Errorf is wrapper for logrus.Errorf to print to stderr
func Errorf(format string, args ...interface{}) {
	stderrLogger.Error(safeMessage(fmt.Sprintf(format, args...)))
}

// Fatalf is wrapper for logrus.Fatalf to print to stderr
func Fatalf(format string, args ...interface{}) {
	stderrLogger.Fatal(safeMessage(fmt.Sprintf(format, args...)))
}

// Warnf is wrapper for logrus.Warnf to print to stderr
func Warnf(format string, args ...interface{}) {
	stderrLogger.Warn(safeMessage(fmt.Sprintf(format, args...)))
}

func safeMessage(message string) string {
	message = strings.ReplaceAll(message, "\r", "")
	return strings.ReplaceAll(message, "\n", " ")
}

// ParseLevel takes a string level and returns the Logrus log level constant.
func ParseLevel(lvl string) (logrus.Level, error) {
	return logrus.ParseLevel(lvl)
}

// SetLevelString takes in the log level in string format
// some of valid values: error, info, debug ...
func SetLevelString(lvlStr string) error {
	level, err := ParseLevel(lvlStr)
	if err != nil {
		return err
	}
	SetLevel(level)
	return nil
}

// SetLevel sets the log level
func SetLevel(lvl logrus.Level) {
	stdoutLogger.Level = lvl
	stderrLogger.Level = lvl
}

// GetLevel gets the current log level
func GetLevel() logrus.Level {
	return stdoutLogger.Level
}

// GetLevelString gets the current log level
func GetLevelString() string {
	var level string
	switch stdoutLogger.Level {
	case logrus.DebugLevel:
		level = "debug"
	case logrus.InfoLevel:
		level = "info"
	case logrus.ErrorLevel:
		level = "error"
	}

	return level
}
