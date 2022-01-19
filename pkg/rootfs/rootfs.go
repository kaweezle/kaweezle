/*
Copyright © 2022 Antoine Martin <antoine@openance.com>

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
package rootfs

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitfield/script"
	"github.com/dustin/go-humanize"
	"github.com/pterm/pterm"
	log "github.com/sirupsen/logrus"
)

const (
	HomeDirName       = "kaweezle"
	TarFilename       = "rootfs.tar.gz"
	RootFSURL         = "https://github.com/kaweezle/kaweezle-rootfs/releases/download/latest/" + TarFilename
	RootFSChecksumURL = RootFSURL + ".sha256"
)

var (
	HomeDir             = filepath.Join(os.Getenv("LOCALAPPDATA"), HomeDirName)
	DefaultTarFilePath  = filepath.Join(HomeDir, TarFilename)
	TarFilePath         = DefaultTarFilePath
	TarFileChecksumPath = TarFilePath + ".sha256"
)

func EnsureHomeDir(homeDir string) (err error) {
	err = os.MkdirAll(homeDir, os.ModePerm)
	return
}

func getReleaseChecksum() (checksum string, err error) {
	var resp *http.Response
	if resp, err = http.DefaultClient.Get(RootFSChecksumURL); err != nil {
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var bodyBytes []byte
		if bodyBytes, err = io.ReadAll(resp.Body); err == nil {
			checksum = strings.TrimSpace(string(bodyBytes))
		}
	}

	return
}

type WritableProgress struct {
	*pterm.ProgressbarPrinter
}

func (wp *WritableProgress) Write(p []byte) (n int, err error) {
	n = len(p)
	wp.Add(n)
	return
}

func EnsureRootFS(path string, fields *log.Fields) (err error) {
	var tarFilePath string
	if tarFilePath, err = filepath.Abs(path); err != nil {
		return
	}
	homeDir := filepath.Dir(tarFilePath)
	if err = EnsureHomeDir(homeDir); err != nil {
		log.WithError(err).WithFields(*fields).Debug("Home directory")
		return
	}

	// Internal error
	var _err error
	current := script.File(tarFilePath)
	currentExists := script.IfExists(tarFilePath).Error() == nil
	tarFileChecksumPath := tarFilePath + ".sha256"

	var currentChecksum, onlineChecksum string
	if currentExists {
		// Get current checksum, either by reading file or by computing checksum
		if currentChecksum, _err = script.File(tarFileChecksumPath).String(); _err != nil {
			if currentChecksum, err = current.SHA256Sum(); err != nil {
				log.WithError(err).Debug("Getting current root fs checksum")
				return
			} else {
				if _, err = script.Echo(currentChecksum).WriteFile(tarFileChecksumPath); err != nil {
					return
				}
			}
		}
	}

	log.WithFields(log.Fields{
		"rootFS":   tarFilePath,
		"exists":   currentExists,
		"checksum": currentChecksum,
	}).Info("Root FS exists: ", currentExists)

	if onlineChecksum, err = getReleaseChecksum(); err != nil {
		return
	}

	log.WithFields(log.Fields{
		"currentChecksum": currentChecksum,
		"onlineChecksum":  onlineChecksum,
	}).Trace("Checksums")

	if onlineChecksum == currentChecksum {
		log.WithFields(log.Fields{
			"rootFS":   tarFilePath,
			"checksum": onlineChecksum,
		}).Info("Root FS already up to date")
		return
	}

	var resp *http.Response

	log.WithFields(log.Fields{
		"rootFS":    tarFilePath,
		"checksum":  onlineChecksum,
		"rootFSURL": RootFSURL,
	}).Info("Donloading Root FS")

	if resp, err = http.DefaultClient.Get(RootFSURL); err != nil {
		return
	}

	defer resp.Body.Close()

	var rootFSTemp *os.File
	if rootFSTemp, err = ioutil.TempFile(homeDir, "rootfs"); err != nil {
		return
	}
	defer rootFSTemp.Close()
	defer os.Remove(rootFSTemp.Name())

	var bar *pterm.ProgressbarPrinter
	title := fmt.Sprintf("%s: %s", filepath.Base(tarFilePath), humanize.Bytes(uint64(resp.ContentLength)))
	if bar, err = pterm.DefaultProgressbar.WithShowCount(false).WithShowElapsedTime(true).WithShowPercentage(true).WithTitle(title).WithTotal(int(resp.ContentLength)).Start(); err != nil {
		return
	}

	bar.Start()
	defer bar.Stop()

	hasher := sha256.New()
	if _, err = io.Copy(io.MultiWriter(rootFSTemp, &WritableProgress{bar}, hasher), resp.Body); err != nil {
		return
	}

	rootFSTemp.Close()
	bar.Stop()

	downloadedChecksum := fmt.Sprintf("%x", hasher.Sum(nil))

	log.WithFields(log.Fields{
		"downloadedChecksum": downloadedChecksum,
		"onlineChecksum":     onlineChecksum,
	}).Trace("Checksums")

	if downloadedChecksum != onlineChecksum {
		err = fmt.Errorf("bad checksum for url %s. Expected %s, got %s", RootFSURL, onlineChecksum, downloadedChecksum)
		return
	}

	if currentExists {
		os.Remove(tarFilePath)
	}

	os.Rename(rootFSTemp.Name(), tarFilePath)

	_, err = script.Echo(downloadedChecksum).WriteFile(tarFileChecksumPath)

	log.WithFields(log.Fields{
		"rootFS":   tarFilePath,
		"checksum": onlineChecksum,
	}).Info("Download ok")

	return
}
