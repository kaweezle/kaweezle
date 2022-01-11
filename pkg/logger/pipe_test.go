package logger

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	ht "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	basicLog = []byte(`{"level":"info","msg":"Starting openrc...","time":"2022-01-11T14:44:25Z"}`)

	basicExecution = `{"level":"info","msg":"Starting openrc...","time":"2022-01-11T14:44:25Z"}
{"level":"info","msg":"Starting the kubelet service...","time":"2022-01-11T14:44:27Z"}
{"level":"info","msg":"Waiting for service to start...","time":"2022-01-11T14:44:27Z"}
{"level":"info","msg":"Apply base kustomization...","time":"2022-01-11T14:44:39Z"}
{"level":"info","msg":"executed","time":"2022-01-11T14:44:41Z"}
{"msg":"Starting openrc...","time":"2022-01-11T14:44:25Z"}
no json
`
)

func TestBasicPipe(t *testing.T) {

	parsedLogEntry := make(ParsedLogEntry)

	require.NoError(t, json.Unmarshal(basicLog, &parsedLogEntry))
	entry, err := parsedLogEntry.LogEntry()
	require.NoError(t, err)

	assert.True(t, entry.Level == logrus.InfoLevel, "Bad log level")
	assert.Equal(t, "Starting openrc...", entry.Message, "Bad message")
	assert.False(t, entry.Time.IsZero())
	assert.Empty(t, entry.Data)
}

func TestBasicPipeWithFields(t *testing.T) {

	_log := []byte(`{"level":"info","msg":"Starting openrc...","time":"2022-01-11T14:44:25Z","toto":"tata"}`)
	parsedLogEntry := make(ParsedLogEntry)

	require.NoError(t, json.Unmarshal(_log, &parsedLogEntry))
	entry, err := parsedLogEntry.LogEntry()
	require.NoError(t, err)

	assert.True(t, entry.Level == logrus.InfoLevel, "Bad log level")
	assert.Equal(t, "Starting openrc...", entry.Message, "Bad message")
	assert.False(t, entry.Time.IsZero())
	assert.NotEmpty(t, entry.Data)
	assert.Contains(t, entry.Data, "toto")
	assert.Equal(t, entry.Data["toto"], "tata")
}

func TestWithNoTime(t *testing.T) {
	_log := []byte(`{"level":"info","msg":"Starting openrc..."}`)
	parsedLogEntry := make(ParsedLogEntry)

	require.NoError(t, json.Unmarshal(_log, &parsedLogEntry))
	_, err := parsedLogEntry.LogEntry()
	require.Error(t, err, "No error on time")
	require.Equal(t, "there is no time string", err.Error())

}

func TestWithBadTimeType(t *testing.T) {
	_log := []byte(`{"level":"info","msg":"Starting openrc...", "time":12}`)
	parsedLogEntry := make(ParsedLogEntry)

	require.NoError(t, json.Unmarshal(_log, &parsedLogEntry))
	_, err := parsedLogEntry.LogEntry()
	require.Error(t, err, "No error on time")
	require.Equal(t, "bad type for time: 12", err.Error())

}

func TestWithBadTimeString(t *testing.T) {
	_log := []byte(`{"level":"info","msg":"Starting openrc...", "time":"tata"}`)
	parsedLogEntry := make(ParsedLogEntry)

	require.NoError(t, json.Unmarshal(_log, &parsedLogEntry))
	_, err := parsedLogEntry.LogEntry()
	require.Error(t, err, "No error on time")
	require.Equal(t, "bad time string: tata", err.Error())

}

func TestWithNoLevel(t *testing.T) {
	_log := []byte(`{"msg":"Starting openrc...","time":"2022-01-11T14:44:25Z"}`)
	parsedLogEntry := make(ParsedLogEntry)

	require.NoError(t, json.Unmarshal(_log, &parsedLogEntry))
	_, err := parsedLogEntry.LogEntry()
	require.Error(t, err, "No error on time")
	require.Equal(t, "no level in entry", err.Error())

}

func TestWithBadLevel(t *testing.T) {
	_log := []byte(`{"level": "toto","msg":"Starting openrc...","time":"2022-01-11T14:44:25Z"}`)
	parsedLogEntry := make(ParsedLogEntry)

	require.NoError(t, json.Unmarshal(_log, &parsedLogEntry))
	_, err := parsedLogEntry.LogEntry()
	require.Error(t, err, "No error on time")
	require.Equal(t, "unkown log level: toto", err.Error())

}

func TestWithBadLevelType(t *testing.T) {
	_log := []byte(`{"level": 12,"msg":"Starting openrc...","time":"2022-01-11T14:44:25Z"}`)
	parsedLogEntry := make(ParsedLogEntry)

	require.NoError(t, json.Unmarshal(_log, &parsedLogEntry))
	_, err := parsedLogEntry.LogEntry()
	require.Error(t, err, "No error on time")
	require.Equal(t, "bad type for level: 12", err.Error())

}

func TestWithNoMessage(t *testing.T) {
	_log := []byte(`{"level": "info","time":"2022-01-11T14:44:25Z"}`)
	parsedLogEntry := make(ParsedLogEntry)

	require.NoError(t, json.Unmarshal(_log, &parsedLogEntry))
	_, err := parsedLogEntry.LogEntry()
	require.Error(t, err, "No error on time")
	require.Equal(t, "there is no message entry", err.Error())

}

func TestWithBadMessageType(t *testing.T) {
	_log := []byte(`{"level": "info","msg": 12,"time":"2022-01-11T14:44:25Z"}`)
	parsedLogEntry := make(ParsedLogEntry)

	require.NoError(t, json.Unmarshal(_log, &parsedLogEntry))
	_, err := parsedLogEntry.LogEntry()
	require.Error(t, err, "No error on time")
	require.Equal(t, "bad message type: 12", err.Error())

}

func TestPipe(t *testing.T) {
	r := strings.NewReader(basicExecution)

	h := ht.NewGlobal()
	logrus.StandardLogger().SetOutput(ioutil.Discard)

	PipeLogs(r, logrus.Fields{})
	require.NotEmpty(t, h.Entries)
	require.Equal(t, 7, len(h.Entries))
}
