package logger

import (
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/kyokomi/emoji"
	"github.com/pterm/pterm"
	log "github.com/sirupsen/logrus"
)

const TaskKey = "task"

type Log struct {
	*log.Logger
}

func (l *Log) InitFileLogging(p string, json bool) *Log {
	var err error

	logFile, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY, 0644)
	if json {
		l.SetFormatter(&log.JSONFormatter{})
	} else {
		l.SetFormatter(&log.TextFormatter{DisableColors: true})
	}
	if err != nil {
		pterm.Fatal.Printfln("Failed to open log file %s for output: %s", p, err)
	}
	l.SetOutput(logFile)
	log.RegisterExitHandler(func() {
		if logFile == nil {
			return
		}
		logFile.Close()
	})
	return l
}

func StandardLogger() *Log {
	return &Log{log.StandardLogger()}
}

func InitFileLogging(p string, json bool) *Log {
	return StandardLogger().InitFileLogging(p, json)
}

type PTermFormatter struct {
	Emoji bool
}

func LevelPrinter(l log.Level) (p pterm.PrefixPrinter) {
	switch l {

	case log.PanicLevel:
		p = pterm.Fatal
	case log.FatalLevel:
		p = pterm.Fatal
	case log.ErrorLevel:
		p = pterm.Error
	case log.WarnLevel:
		p = pterm.Warning
	case log.InfoLevel:
		p = pterm.Info
	case log.DebugLevel:
		p = pterm.Debug
	case log.TraceLevel:
		p = pterm.Description
	}
	return
}

var (
	spinnerLock sync.Mutex
	spinner     = pterm.DefaultSpinner.WithShowTimer(true).WithRemoveWhenDone(false)
	spinners    = make(map[string](*pterm.SpinnerPrinter))
)

var emojiStrip = regexp.MustCompile(`[:][\w]+[:]`)

func joinMsg(args ...interface{}) (message string) {
	for _, m := range args {
		message += " " + fmt.Sprintf("%v", m)
	}
	return
}

func (f *PTermFormatter) transform(a string) (result string) {

	// Strip emoji if needed
	if f.Emoji {
		result = emoji.Sprint(a)
	} else {
		result = emojiStrip.ReplaceAllString(joinMsg(a), "")
	}

	return
}

func (f *PTermFormatter) Format(entry *log.Entry) (b []byte, err error) {

	b = []byte{}

	transformed := f.transform(entry.Message)

	if rawTask, ok := entry.Data[TaskKey]; ok {
		if task, ok := rawTask.(string); ok {
			spinnerLock.Lock()
			defer spinnerLock.Unlock()

			currentSpinner, exists := spinners[task]

			if rawError, ok := entry.Data[log.ErrorKey]; ok && exists {
				if err, ok := rawError.(error); ok {
					currentSpinner.Fail(err.Error())
				} else {
					currentSpinner.Success(transformed)
				}
				delete(spinners, task)
			} else {
				if !exists {
					currentSpinner, _ = spinner.Start(task)
					spinners[task] = currentSpinner
				}
				currentSpinner.UpdateText(fmt.Sprintf("%s  ➜  %s", task, transformed))
			}
		}
	} else {
		printer := LevelPrinter(entry.Level)
		printer.Println(transformed)
	}

	return
}
