// Copyright © 2021 Ettore Di Giacinto <mudler@mocaccino.org>
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <http://www.gnu.org/licenses/>.

package logger

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"

	"github.com/kyokomi/emoji"
	"github.com/pterm/pterm"
)

var (
	std = New()
)

// Logger is the default logger
type Logger struct {
	level       log.Level
	emoji       bool
	logToFile   bool
	noSpinner   bool
	fileLogger  *log.Logger
	context     string
	spinnerLock sync.Mutex
	s           *pterm.SpinnerPrinter
}

// LogLevel represents a log severity level. Use the package variables as an
// enum.
type LogLevel log.Level

func InitTerm() {
	pterm.Info.Prefix = pterm.Prefix{
		Text:  "➜",
		Style: pterm.NewStyle(pterm.FgGreen),
	}
	pterm.Warning.Prefix = pterm.Prefix{
		Text: emoji.Sprint(":warning:"),
	}
}

func (l *Logger) WithoutSpinner() *Logger {
	l.noSpinner = true
	return l
}

func WithoutSpinner() {
	std.WithoutSpinner()
}

func (l *Logger) WithLevel(level string) *Logger {
	lvl, _ := log.ParseLevel(level) // Defaults to Info
	l.level = lvl
	if l.level == log.DebugLevel {
		pterm.EnableDebugMessages()
	}
	return l
}

func WithLevel(level string) {
	std.WithLevel(level)
}

func (l *Logger) WithContext(c string) *Logger {
	l.context = c
	return l
}

func WithContext(c string) {
	std.WithContext(c)
}

func (l *Logger) WithFileLogging(p string, json bool) *Logger {
	l.logToFile = true
	var err error

	l.fileLogger = log.New()
	l.fileLogger.SetLevel(log.DebugLevel)
	logFile, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY, 0644)
	if json {
		l.fileLogger.SetFormatter(&log.JSONFormatter{})
	} else {
		l.fileLogger.SetFormatter(&log.TextFormatter{DisableColors: true})
	}
	if err != nil {
		pterm.Fatal.Printfln("Failed to open log file %s for output: %s", p, err)
	}
	l.fileLogger.SetOutput(logFile)
	log.RegisterExitHandler(func() {
		if logFile == nil {
			return
		}
		logFile.Close()
	})
	return l
}

func WithFileLogging(p string, json bool) {
	std.WithFileLogging(p, json)
}

func (l *Logger) WithEmoji() *Logger {
	l.emoji = true
	return l
}

func WithEmoji() {
	std.WithEmoji()
}

func New() *Logger {
	return &Logger{
		level: log.InfoLevel,
		emoji: true,
		s:     pterm.DefaultSpinner.WithShowTimer(false).WithRemoveWhenDone(true),
	}
}

func (l *Logger) Copy() *Logger {
	c := *l
	copy := &c

	return copy
}

func joinMsg(args ...interface{}) (message string) {
	for _, m := range args {
		message += " " + fmt.Sprintf("%v", m)
	}
	return
}

func (l *Logger) enabled(lvl log.Level) bool {
	return lvl <= l.level
}

var emojiStrip = regexp.MustCompile(`[:][\w]+[:]`)

func (l *Logger) transform(args ...interface{}) (sanitized []interface{}) {
	for _, a := range args {
		var aString string

		// Strip emoji if needed
		if l.emoji {
			aString = emoji.Sprint(a)
		} else {
			aString = emojiStrip.ReplaceAllString(joinMsg(a), "")
		}

		sanitized = append(sanitized, aString)
	}

	if l.context != "" {
		sanitized = append([]interface{}{fmt.Sprintf("(%s)", l.context)}, sanitized...)
	}
	return
}

func prefixCodeLine(args ...interface{}) []interface{} {
	pc, file, line, ok := runtime.Caller(3)
	if ok {
		args = append([]interface{}{fmt.Sprintf("(%s:#%d:%v)",
			path.Base(file), line, runtime.FuncForPC(pc).Name())}, args...)
	}
	return args
}

func (l *Logger) send(ll log.Level, f string, args ...interface{}) {
	if !l.enabled(ll) {
		return
	}

	sanitizedArgs := joinMsg(l.transform(args...)...)
	sanitizedF := joinMsg(l.transform(f)...)
	formatDefined := f != ""

	switch {
	case log.DebugLevel == ll && !formatDefined:
		pterm.Debug.Println(prefixCodeLine(sanitizedArgs)...)
		if l.logToFile {
			l.fileLogger.Debug(joinMsg(prefixCodeLine(sanitizedArgs)...))
		}
	case log.DebugLevel == ll && formatDefined:
		pterm.Debug.Printfln(sanitizedF, prefixCodeLine(args...)...)
		if l.logToFile {
			l.fileLogger.Debugf(sanitizedF, prefixCodeLine(args...)...)
		}
	case log.ErrorLevel == ll && !formatDefined:
		pterm.Error.Println(pterm.LightRed(sanitizedArgs))
		if l.logToFile {
			l.fileLogger.Error(sanitizedArgs)
		}
	case log.ErrorLevel == ll && formatDefined:
		pterm.Error.Printfln(pterm.LightRed(sanitizedF), args...)
		if l.logToFile {
			l.fileLogger.Errorf(sanitizedF, args...)
		}

	case log.FatalLevel == ll && !formatDefined:
		pterm.Error.Println(sanitizedArgs)
		if l.logToFile {
			l.fileLogger.Error(sanitizedArgs)
		}
	case log.FatalLevel == ll && formatDefined:
		pterm.Error.Printfln(sanitizedF, args...)
		if l.logToFile {
			l.fileLogger.Errorf(sanitizedF, args...)
		}
		//INFO
	case log.InfoLevel == ll && !formatDefined:
		pterm.Info.Println(sanitizedArgs)
		if l.logToFile {
			l.fileLogger.Info(sanitizedArgs)
		}
	case log.InfoLevel == ll && formatDefined:
		pterm.Info.Printfln(sanitizedF, args...)
		if l.logToFile {
			l.fileLogger.Infof(sanitizedF, args...)
		}
		//WARN
	case log.WarnLevel == ll && !formatDefined:
		pterm.Warning.Println(sanitizedArgs)
		if l.logToFile {
			l.fileLogger.Warn(sanitizedArgs)
		}
	case log.WarnLevel == ll && formatDefined:
		pterm.Warning.Printfln(sanitizedF, args...)
		if l.logToFile {
			l.fileLogger.Warnf(sanitizedF, args...)
		}
	}
}

func (l *Logger) Debug(args ...interface{}) {
	l.send(log.DebugLevel, "", args...)
}

func (l *Logger) Error(args ...interface{}) {
	l.send(log.ErrorLevel, "", args...)
}

func (l *Logger) Trace(args ...interface{}) {
	l.send(log.DebugLevel, "", args...)
}

func (l *Logger) Tracef(t string, args ...interface{}) {
	l.send(log.DebugLevel, t, args...)
}

func (l *Logger) Fatal(args ...interface{}) {
	l.send(log.FatalLevel, "", args...)
	os.Exit(1)
}

func (l *Logger) Info(args ...interface{}) {
	l.send(log.InfoLevel, "", args...)
}

func (l *Logger) Success(args ...interface{}) {
	l.Info(append([]interface{}{"SUCCESS"}, args...)...)
}

func (l *Logger) Panic(args ...interface{}) {
	l.Fatal(args...)
}

func (l *Logger) Warn(args ...interface{}) {
	l.send(log.WarnLevel, "", args...)
}

func (l *Logger) Warning(args ...interface{}) {
	l.Warn(args...)
}

func (l *Logger) Debugf(f string, args ...interface{}) {
	l.send(log.DebugLevel, f, args...)
}

func (l *Logger) Errorf(f string, args ...interface{}) {
	l.send(log.ErrorLevel, f, args...)
}

func (l *Logger) Fatalf(f string, args ...interface{}) {
	l.send(log.FatalLevel, f, args...)
}

func (l *Logger) Infof(f string, args ...interface{}) {
	l.send(log.InfoLevel, f, args...)
}

func (l *Logger) Panicf(f string, args ...interface{}) {
	l.Fatalf(joinMsg(f), args...)
}

func (l *Logger) Warnf(f string, args ...interface{}) {
	l.send(log.WarnLevel, f, args...)
}

func (l *Logger) Warningf(f string, args ...interface{}) {
	l.Warnf(f, args...)
}

func (l *Logger) Ask() bool {
	var input string

	l.Info("Do you want to continue with this operation? [y/N]: ")
	_, err := fmt.Scanln(&input)
	if err != nil {
		return false
	}
	input = strings.ToLower(input)

	if input == "y" || input == "yes" {
		return true
	}
	return false
}

func IsTerminal() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}

// Spinner starts the spinner
func (l *Logger) Spinner() {
	if !IsTerminal() || l.noSpinner {
		return
	}

	l.spinnerLock.Lock()
	defer l.spinnerLock.Unlock()

	if l.s != nil && !l.s.IsActive {
		l.s, _ = l.s.Start()
	}
}

func (l *Logger) Screen(text string) {
	pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgLightBlue)).WithMargin(2).Println(text)
}

func (l *Logger) SpinnerText(suffix, prefix string) {
	if !IsTerminal() || l.noSpinner {
		return
	}
	l.spinnerLock.Lock()
	defer l.spinnerLock.Unlock()

	if l.level == log.DebugLevel {
		fmt.Printf("%s %s\n",
			suffix, prefix,
		)
	} else {
		l.s.UpdateText(suffix + prefix)
	}
}

func (l *Logger) SpinnerStop() {
	if !IsTerminal() || l.noSpinner {
		return
	}
	l.spinnerLock.Lock()
	defer l.spinnerLock.Unlock()

	if l.s != nil {
		l.s.Success()
	}
}

/////////////////

func Debug(args ...interface{}) {
	std.Debug(args...)
}

func Error(args ...interface{}) {
	std.Error(args...)
}

func Trace(args ...interface{}) {
	std.Trace(args...)
}

func Tracef(t string, args ...interface{}) {
	std.Tracef(t, args...)
}

func Fatal(args ...interface{}) {
	std.Fatal(args...)
}

func Info(args ...interface{}) {
	std.Info(args...)
}

func Success(args ...interface{}) {
	std.Success(args...)
}

func Panic(args ...interface{}) {
	std.Panic(args...)
}

func Warn(args ...interface{}) {
	std.Warn(args...)
}

func Warning(args ...interface{}) {
	std.Warning(args...)
}

func Debugf(f string, args ...interface{}) {
	std.Debugf(f, args...)
}

func Errorf(f string, args ...interface{}) {
	std.Errorf(f, args...)
}

func Fatalf(f string, args ...interface{}) {
	std.Fatalf(f, args...)
}

func Infof(f string, args ...interface{}) {
	std.Infof(f, args...)
}

func Panicf(f string, args ...interface{}) {
	std.Panicf(f, args...)
}

func Warnf(f string, args ...interface{}) {
	std.Warnf(f, args...)
}

func Warningf(f string, args ...interface{}) {
	std.Warningf(f, args...)
}

func Ask() bool {
	return std.Ask()
}

// Spinner starts the spinner
func Spinner() {
	std.Spinner()
}

func Screen(text string) {
	std.Screen(text)
}

func SpinnerText(suffix, prefix string) {
	std.SpinnerText(suffix, prefix)
}

func SpinnerStop() {
	std.SpinnerStop()
}
