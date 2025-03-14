/*
Copyright © 2021 Antoine Martin <antoine@openance.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package logger

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"sort"
	"sync"
	"unicode"

	"github.com/kyokomi/emoji"
	"github.com/pterm/pterm"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

const (
	TaskKey        = "task"
	FieldsPrefix   = "└  "
	QuoteCharacter = "\""
)

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
	Emoji      bool
	ShowFields bool
}

type Prefix struct {
	Text  string
	Style *pterm.Style
}

const (
	SuccessLevel log.Level = log.TraceLevel + 1
)

var LevelPrefixes = map[log.Level]*Prefix{
	log.PanicLevel: {"💣", pterm.NewStyle(pterm.FgLightRed)},
	log.FatalLevel: {"💣", pterm.NewStyle(pterm.FgLightRed)},
	log.ErrorLevel: {"🐞", pterm.NewStyle(pterm.FgRed)},
	log.WarnLevel:  {"⚡", pterm.NewStyle(pterm.FgYellow)},
	log.InfoLevel:  {"🚀", pterm.NewStyle(pterm.FgLightCyan)},
	log.DebugLevel: {"💬", pterm.NewStyle(pterm.FgLightMagenta)},
	log.TraceLevel: {"👀", pterm.NewStyle(pterm.FgWhite)},
	SuccessLevel:   {"🍺", pterm.NewStyle(pterm.FgLightGreen)},
}

var clocks = []string{"🕐", "🕑", "🕒", "🕓", "🕔", "🕕", "🕖", "🕗", "🕘", "🕙", "🕚", "🕛"}

// var braille = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
var (
	spinnerLock sync.Mutex
	spinner     = pterm.DefaultSpinner.WithShowTimer(false).WithRemoveWhenDone(true).WithSequence(clocks...)
	spinners    = make(map[string](*pterm.SpinnerPrinter))
)

var emojiStrip = regexp.MustCompile(`[:][\w]+[:]`)

func joinMsg(args ...interface{}) (message string) {
	for _, m := range args {
		message += fmt.Sprintf("%v", m) + " "
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

func ellipsize(str string, max int) string {
	lastSpaceIx := -1
	len := 0
	for i, r := range str {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			lastSpaceIx = i
		}
		if unicode.IsPrint(r) {
			len++
		}
		if len >= max {
			if lastSpaceIx != -1 {
				return str[:lastSpaceIx] + "..."
			}
		}
	}
	return str
}

func (f *PTermFormatter) FormatFields(data log.Fields, l int, flp string, style *pterm.Style) string {
	b := &bytes.Buffer{}

	var keys []string = lo.Keys(data)
	sort.Strings(keys)

	ll := 0
	flpLength := len(flp)
	for i, key := range keys {

		bb := &bytes.Buffer{}
		f.appendKeyValue(bb, key, data[key])
		vl := len(bb.String())
		bb.Reset()
		f.appendKeyValue(bb, style.Sprint(key), data[key])
		if i > 0 {
			if ll+vl >= l {
				b.WriteByte('\n')
				b.WriteString(flp)
				ll = vl + flpLength
			} else {
				b.WriteByte(' ')
				ll += 1 + vl
			}
		} else {
			ll = vl
		}
		b.WriteString(bb.String())
	}
	return b.String()
}

func (f *PTermFormatter) needsQuoting(text string) bool {
	if len(text) == 0 {
		return true
	}
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.') {
			return true
		}
	}
	return false
}

func (f *PTermFormatter) appendValue(b *bytes.Buffer, value interface{}) {
	switch value := value.(type) {
	case string:
		if !f.needsQuoting(value) {
			b.WriteString(value)
		} else {
			fmt.Fprintf(b, "%s%v%s", QuoteCharacter, value, QuoteCharacter)
		}
	case error:
		errorMessage := value.Error()
		if !f.needsQuoting(errorMessage) {
			b.WriteString(errorMessage)
		} else {
			fmt.Fprintf(b, "%s%v%s", QuoteCharacter, errorMessage, QuoteCharacter)
		}
	default:
		fmt.Fprint(b, value)
	}
}

func (f *PTermFormatter) appendKeyValue(b *bytes.Buffer, key string, value interface{}) {
	b.WriteString(key)
	b.WriteByte('=')
	f.appendValue(b, value)
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
				prefix := LevelPrefixes[SuccessLevel]

				if err, ok := rawError.(error); ok {
					transformed = err.Error()
					prefix = LevelPrefixes[log.ErrorLevel]
				}
				currentSpinner.Stop()
				pterm.Println(prefix.Style.Sprint(prefix.Text), prefix.Style.Sprint(transformed))
				delete(spinners, task)
			} else {
				if !exists {
					currentSpinner, _ = spinner.Start(task)
					spinners[task] = currentSpinner
				}
				text := fmt.Sprintf("%s  ➜  %s", task, transformed)
				ellipsized := ellipsize(text, pterm.GetTerminalWidth()-5)
				currentSpinner.UpdateText(ellipsized)
			}
		}
	} else {
		prefix := LevelPrefixes[entry.Level]
		pterm.Println(prefix.Style.Sprint(prefix.Text), prefix.Style.Sprint(transformed))
		fields := lo.OmitBy(entry.Data, func(k string, v any) bool {
			return k == "error" && v == nil
		})
		if f.ShowFields && len(fields) > 0 {
			pterm.Println(pterm.Gray(FieldsPrefix, f.FormatFields(fields, pterm.GetTerminalWidth(), "   ", prefix.Style)))
		}
	}

	return
}
