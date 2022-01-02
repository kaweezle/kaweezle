package wsl

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/text/encoding/unicode"

	"github.com/antoinemartin/kaweezle/pkg/logger"
	log "github.com/sirupsen/logrus"
	"github.com/yuk7/wsllib-go"
)

type DistributionState int16

const (
	Unknown DistributionState = iota
	Stopped
	Running
)

func (s DistributionState) String() (r string) {
	switch s {
	case Unknown:
		r = "unknown"
	case Stopped:
		r = "Stopped"
	case Running:
		r = "Running"
	}
	return
}

func ParseDistributionState(label string) (s DistributionState, err error) {
	switch label {
	case "Stopped":
		s = Stopped
	case "Running":
		s = Running
	default:
		s = Unknown
		err = fmt.Errorf("unknown distribution state: %v", label)
	}
	return
}

type DistributionInformation struct {
	Name      string
	State     DistributionState
	Version   int
	IsDefault bool
}

func GetDistributions() (result map[string]DistributionInformation, err error) {

	result = make(map[string]DistributionInformation, 0)
	if out, err := exec.Command("C:\\Windows\\system32\\wsl.exe", "--list", "--verbose").Output(); err == nil {
		enc := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
		out, _ = enc.NewDecoder().Bytes(out)

		log.WithField("out", string(out)).Trace("WSL output")
		lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
		log.WithField("lineCount", len(lines)).Trace("Lines")
		for _, line := range lines[1 : len(lines)-1] {
			fields := strings.Fields(line)
			log.WithFields(log.Fields{
				"fields":     fields,
				"fieldCount": len(fields),
			}).Trace("Line fields")
			isDefault := false
			if len(fields) == 4 {
				isDefault = true
				fields = fields[1:]
			}
			name := fields[0]
			var state DistributionState
			if state, err = ParseDistributionState(fields[1]); err == nil {
				var version int
				if version, err = strconv.Atoi(fields[2]); err == nil {
					info := DistributionInformation{
						Name:      name,
						State:     state,
						Version:   version,
						IsDefault: isDefault,
					}
					log.WithField("distribution", info).Trace("Appending information")
					result[name] = info
				} else {
					log.WithError(err).WithField("distribution_name", name).Warning("Error while conveting WSL version")
				}
			} else {
				log.WithError(err).WithField("distribution_name", name).Trace("Error while parsing distribution state")
			}

			if err != nil {
				break
			}
		}
	} else {
		log.WithError(err).Error("WSL error")
	}

	log.WithField("distributions", result).Trace("result")

	return
}

func GetDistribution(name string) (info DistributionInformation, err error) {
	var distributions map[string]DistributionInformation
	if distributions, err = GetDistributions(); err == nil {
		info = distributions[name]
		log.WithFields(log.Fields{
			"result":            info,
			"distribution_name": name,
		}).Trace("result")
	}
	return
}

func StopDistribution(name string) (err error) {
	var out []byte
	if out, err = exec.Command("C:\\Windows\\system32\\wsl.exe", "--terminate", name).Output(); err == nil {
		enc := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
		out, _ = enc.NewDecoder().Bytes(out)
		log.WithFields(log.Fields{
			"output":            out,
			"distribution_name": name,
		}).Trace("result")
	}

	return
}

func LaunchAndPipe(distributionName string, command string, useCurrentWorkingDirectory bool, fields log.Fields) (exitCode uint32, err error) {
	p, _ := syscall.GetCurrentProcess()

	rout, wout, _ := os.Pipe()

	stdin := syscall.Handle(0)
	stdout := syscall.Handle(0)
	stderr := syscall.Handle(0)

	syscall.DuplicateHandle(p, syscall.Handle(os.Stdin.Fd()), p, &stdin, 0, true, syscall.DUPLICATE_SAME_ACCESS)
	syscall.DuplicateHandle(p, syscall.Handle(os.Stdout.Fd()), p, &stdout, 0, true, syscall.DUPLICATE_SAME_ACCESS)
	syscall.DuplicateHandle(p, syscall.Handle(wout.Fd()), p, &stderr, 0, true, syscall.DUPLICATE_SAME_ACCESS)

	log.WithFields(log.Fields{
		"command":           command,
		"distribution_name": distributionName,
	}).Debug("Start WSL command")
	handle, err := wsllib.WslLaunch(distributionName, command, useCurrentWorkingDirectory, stdin, stdout, stderr)
	// No more needed
	wout.Close()
	syscall.CloseHandle(stderr)
	logger.PipeLogs(rout, fields)

	syscall.WaitForSingleObject(handle, syscall.INFINITE)
	syscall.GetExitCodeProcess(handle, &exitCode)
	return
}
