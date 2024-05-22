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
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"time"

	log "github.com/sirupsen/logrus"
)

type ParsedLogEntry map[string]interface{}

func (logEntry *ParsedLogEntry) LogEntry() (entry *log.Entry, err error) {

	var _entry *log.Entry

	if rawTime, ok := (*logEntry)["time"]; ok {
		if timeString, ok := rawTime.(string); ok {
			var dateTime time.Time
			if dateTime, err = time.Parse(time.RFC3339, timeString); err == nil {
				_entry = log.WithTime(dateTime)

				delete(*logEntry, "time")
				if rawLevel, ok := (*logEntry)["level"]; ok {
					if levelString, ok := rawLevel.(string); ok {
						if _entry.Level, err = log.ParseLevel(levelString); err == nil {
							delete(*logEntry, "level")
							if rawMessage, ok := (*logEntry)["msg"]; ok {
								if message, ok := rawMessage.(string); ok {
									_entry.Message = message
									delete(*logEntry, "msg")
									_entry.Data = log.Fields(*logEntry)
									entry = _entry
								} else {
									err = fmt.Errorf("bad message type: %v", rawMessage)
								}
							} else {
								err = fmt.Errorf("there is no message entry")
							}
						} else {
							err = fmt.Errorf("unkown log level: %s", levelString)
						}
					} else {
						err = fmt.Errorf("bad type for level: %v", rawLevel)
					}
				} else {
					err = fmt.Errorf("no level in entry")
				}
			} else {
				err = fmt.Errorf("bad time string: %s", timeString)
			}
		} else {
			err = fmt.Errorf("bad type for time: %v", rawTime)
		}
	} else {
		err = fmt.Errorf("there is no time string")
	}

	return
}

func PipeLogs(rout io.Reader, fields log.Fields) {

	scanner := bufio.NewScanner(rout)
	for scanner.Scan() {
		line := scanner.Bytes()

		parsedLogEntry := make(ParsedLogEntry)

		if err := json.Unmarshal(line, &parsedLogEntry); err == nil {
			if entry, err := parsedLogEntry.LogEntry(); err == nil {
				entry.WithFields(fields).Log(entry.Level, entry.Message)
			} else {
				log.WithError(err).WithField("parsedLog", parsedLogEntry).Warn("Couldn't parse log")
			}
		} else {
			log.WithFields(fields).Info(string(line))
		}
	}
}
