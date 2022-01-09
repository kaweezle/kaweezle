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
	log "github.com/sirupsen/logrus"
)

const (
	TaskKey        = "task"
	FieldsPrefix   = "\n└ "
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

var clocks = []string{"🕐", "🕑", "🕒", "🕓", "🕔", "🕕", "🕖", "🕗", "🕘", "🕙", "🕚", "🕛"}

// var braille = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
var (
	spinnerLock sync.Mutex
	spinner     = pterm.DefaultSpinner.WithShowTimer(true).WithRemoveWhenDone(false).WithSequence(clocks...)
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

func ellipsize(str string, max int) string {
	lastSpaceIx := -1
	len := 0
	for i, r := range str {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			lastSpaceIx = i
		}
		len++
		if len >= max {
			if lastSpaceIx != -1 {
				return str[:lastSpaceIx] + "..."
			}
		}
	}
	return str
}

func (f *PTermFormatter) FormatFields(entry *log.Entry, l int, flp string) string {
	b := &bytes.Buffer{}

	var keys []string = make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ll := 0
	flpl := len(flp)
	for i, key := range keys {

		bb := &bytes.Buffer{}
		f.appendKeyValue(bb, key, entry.Data[key])
		vls := ellipsize(bb.String(), l)
		vl := len(vls)
		if i > 0 {
			if ll+vl >= l {
				b.WriteByte('\n')
				b.WriteString(flp)
				ll = vl + flpl
			} else {
				b.WriteByte(' ')
				ll += 1 + vl
			}
		} else {
			ll = vl
		}
		b.WriteString(vls)
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
		errmsg := value.Error()
		if !f.needsQuoting(errmsg) {
			b.WriteString(errmsg)
		} else {
			fmt.Fprintf(b, "%s%v%s", QuoteCharacter, errmsg, QuoteCharacter)
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
				text := fmt.Sprintf("%s  ➜  %s", task, transformed)
				ellipsized := ellipsize(text, pterm.GetTerminalWidth()-5)
				currentSpinner.UpdateText(ellipsized)
			}
		}
	} else {
		printer := LevelPrinter(entry.Level)
		if f.ShowFields && len(entry.Data) > 0 {
			transformed = transformed + pterm.Gray(FieldsPrefix, f.FormatFields(entry, pterm.GetTerminalWidth()-7, "  "))
		}
		printer.Println(transformed)
	}

	return
}
