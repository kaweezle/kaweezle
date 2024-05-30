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
package wsl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/text/encoding/unicode"

	s "github.com/bitfield/script"
	"github.com/kaweezle/kaweezle/pkg/logger"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/yuk7/wsllib-go"
)

type DistributionState int16

const (
	Unknown DistributionState = iota
	Stopped
	Running
	resolvFilename = "/etc/resolv.conf"
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

var afs = &afero.Afero{Fs: afero.NewOsFs()}

var (
	once    sync.Once
	wslPath string
)

func FindWSL() string {
	// At the time of this writing, a defect appeared in the OS preinstalled WSL executable
	// where it no longer reliably locates the preferred Windows App Store variant.
	//
	// Manually discover (and cache) the wsl.exe location to bypass the problem
	once.Do(func() {
		var locs []string

		// Prefer Windows App Store version
		if LocalAppDirectory := os.Getenv("LOCALAPPDATA"); LocalAppDirectory != "" {
			locs = append(locs, filepath.Join(LocalAppDirectory, "Microsoft", "WindowsApps", "wsl.exe"))
		}

		// Otherwise, the common location for the legacy system version
		root := os.Getenv("SystemRoot")
		if root == "" {
			root = `C:\Windows`
		}
		locs = append(locs, filepath.Join(root, "System32", "wsl.exe"))

		for _, loc := range locs {
			if exists, err := afs.Exists(loc); exists && err == nil {
				wslPath = loc
				return
			}
		}

		// Hope for the best
		wslPath = "wsl"
	})

	return wslPath
}

type DistributionInformation struct {
	Name      string
	State     DistributionState
	Version   int
	IsDefault bool
}

func GetDistributions() (result map[string]DistributionInformation, err error) {

	result = make(map[string]DistributionInformation)
	if out, err := exec.Command(FindWSL(), "--list", "--verbose").Output(); err == nil {
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
					log.WithError(err).WithField("distribution_name", name).Warning("Error while converting WSL version")
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
	if out, err = exec.Command(FindWSL(), "--terminate", name).Output(); err == nil {
		enc := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
		out, _ = enc.NewDecoder().Bytes(out)
		log.WithFields(log.Fields{
			"output":            out,
			"distribution_name": name,
		}).Trace("result")
	}

	return
}

func WslFile(distributionName string, path string) string {
	return fmt.Sprintf(`\\wsl$\%s%s`, distributionName, strings.ReplaceAll(path, "/", `\`))
}

func WslPipe(input string, distributionName string, arg ...string) error {
	newArgs := []string{"-u", "root", "-d", distributionName}
	newArgs = append(newArgs, arg...)
	cmd := exec.Command(FindWSL(), newArgs...)
	cmd.Stdin = strings.NewReader(input)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func WslCommand(distributionName string, arg ...string) ([]byte, error) {
	newArgs := []string{"-u", "root", "-d", distributionName}
	newArgs = append(newArgs, arg...)
	cmd := exec.Command(FindWSL(), newArgs...)
	return cmd.Output()
}

func LaunchAndPipe(distributionName string, command string, useCurrentWorkingDirectory bool, fields log.Fields) (exitCode uint32, err error) {
	p, _ := syscall.GetCurrentProcess()

	rout, writeOut, _ := os.Pipe()

	stdin := syscall.Handle(0)
	stdout := syscall.Handle(0)
	stderr := syscall.Handle(0)

	syscall.DuplicateHandle(p, syscall.Handle(os.Stdin.Fd()), p, &stdin, 0, true, syscall.DUPLICATE_SAME_ACCESS)
	syscall.DuplicateHandle(p, syscall.Handle(os.Stdout.Fd()), p, &stdout, 0, true, syscall.DUPLICATE_SAME_ACCESS)
	syscall.DuplicateHandle(p, syscall.Handle(writeOut.Fd()), p, &stderr, 0, true, syscall.DUPLICATE_SAME_ACCESS)

	log.WithFields(log.Fields{
		"command":           command,
		"distribution_name": distributionName,
	}).Debug("Start WSL command")
	handle, err := wsllib.WslLaunch(distributionName, command, useCurrentWorkingDirectory, stdin, stdout, stderr)
	// No more needed
	writeOut.Close()
	syscall.CloseHandle(stderr)
	logger.PipeLogs(rout, fields)

	syscall.WaitForSingleObject(handle, syscall.INFINITE)
	syscall.GetExitCodeProcess(handle, &exitCode)
	return
}

func RegisterDistribution(name string, rootfs string, path string) (err error) {
	var out []byte
	fields := log.Fields{
		"rootfs":       rootfs,
		"distrib_name": name,
		"install_dir":  path,
		logger.TaskKey: "WSL Registration",
	}

	log.WithFields(fields).Infof("Registering %s in %s from %s", name, path, rootfs)

	if out, err = exec.Command(FindWSL(), "--import", name, path, rootfs).Output(); err == nil {
		enc := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
		out, _ = enc.NewDecoder().Bytes(out)
		log.WithFields(fields).WithField("output", out).Trace("result")
	} else {
		err = fmt.Errorf("error while importing WSL distribution %s in path %s with root file system %s: %v", name, path, rootfs, err)
	}
	log.WithFields(fields).WithError(err).Info("Registration done")

	return
}

func CopyFileToDistribution(distributionName string, source string, destination string, commands ...string) error {

	exist, err := afs.Exists(source)
	if err != nil {
		return errors.Wrapf(err, "error while checking if file %s exists", source)
	}
	if !exist {
		return fmt.Errorf("file %s does not exist", source)
	}

	fields := log.Fields{
		"source":       source,
		"destination":  destination,
		"distrib_name": distributionName,
		logger.TaskKey: "WSL File Copy",
	}

	log.WithFields(fields).Infof("Copying %s to %s in %s", source, destination, distributionName)

	content, err := afs.ReadFile(source)
	if err != nil {
		return errors.Wrapf(err, "error while reading file %s", source)
	}

	err = WslPipe(string(content), distributionName, "sh", "-c", fmt.Sprintf("mkdir -p `dirname '%s'`;", destination)+
		fmt.Sprintf("cat > '%s';", destination)+strings.Join(commands, ";"))

	if err != nil {
		return errors.Wrapf(err, "error while copying file %s to %s in distribution %s", source, destination, distributionName)
	}

	log.WithFields(fields).WithError(err).Infof("Copy of %s to %s in %s done", source, destination, distributionName)

	return nil
}

func FirewallInterface(distributionName string) (string, error) {
	line, err := s.Exec(fmt.Sprintf("wsl -d %s -u root cat %s", distributionName, resolvFilename)).MatchRegexp(regexp.MustCompile(`^nameserver.*$`)).String()
	if err != nil {
		return "", errors.Wrapf(err, "error while reading %s", resolvFilename)
	}
	items := strings.Split(strings.TrimSuffix(line, "\n"), " ")
	if len(items) > 1 {
		return items[1], nil
	} else {
		return "", fmt.Errorf("no nameserver found in %s", resolvFilename)
	}
}
