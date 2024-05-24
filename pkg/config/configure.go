package config

import (
	context "context"
	"fmt"
	"os"
	"regexp"
	"strings"

	s "github.com/bitfield/script"
	"github.com/kaweezle/kaweezle/pkg/wsl"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/txn2/txeh"
)

var afs = &afero.Afero{Fs: afero.NewOsFs()}

func GetAgeKeyFile() string {
	value, exist := os.LookupEnv("SOPS_AGE_KEY_FILE")
	if !exist {
		value = os.Getenv("APPDATA") + "/sops/age/keys.txt"
	}
	return value
}

func ConfigureAgeKeyFile(distributionName string, ageKeyFile string) error {
	if ageKeyFile != "" {
		if exists, _ := afs.Exists(ageKeyFile); !exists {
			log.WithField("age_key_file", ageKeyFile).Warn("Age key file does not exist")
		} else {
			err := wsl.CopyFileToDistribution(distributionName, ageKeyFile, "/root/.config/sops/age/keys.txt")
			if err != nil {
				return errors.Wrap(err, "failed to copy age key file")
			}
			line := "export SOPS_AGE_KEY_FILE=\"/root/.config/sops/age/keys.txt\"\n"
			filename := wsl.WslFile(distributionName, "/etc/conf.d/iknite")
			s.File(filename).RejectRegexp(regexp.MustCompile(`^export SOPS_AGE_KEY_FILE=.*$`)).WriteFile(filename)
			_, err = s.Echo(line).AppendFile(filename)
			return err
		}
	}
	return nil
}

func ConfigureSshKeyFile(distributionName string, sshKeyFile string) error {
	if sshKeyFile != "" {
		if exists, _ := afs.Exists(sshKeyFile); !exists {
			log.WithField("ssh_key_file", sshKeyFile).Warn("SSH key file does not exist")
		} else {
			err := wsl.CopyFileToDistribution(distributionName, sshKeyFile, "/root/.ssh/id_rsa", "chmod 600 /root/.ssh/id_rsa", "chmod 700 /root/.ssh")
			if err != nil {
				return errors.Wrap(err, "failed to copy ssh key file")
			}
		}
	}
	return nil
}

func ConfigureKustomizeUrl(distributionName string, kustomizeUrl string) error {
	if kustomizeUrl != "" {
		log.WithField("kustomize_url", kustomizeUrl).Info("Setting kustomize url...")
		line := fmt.Sprintf("export IKNITE_KUSTOMIZE_DIRECTORY=\"%s\"\n", kustomizeUrl)
		filename := wsl.WslFile(distributionName, "/etc/conf.d/iknite")
		s.File(filename).RejectRegexp(regexp.MustCompile(`^export IKNITE_KUSTOMIZE_DIRECTORY=.*$`)).WriteFile(filename)
		_, err := s.Echo(line).AppendFile(filename)
		return err
	}
	return nil
}

func Configure(distributionName string, options *ConfigurationOptions) error {

	err := ConfigureAgeKeyFile(distributionName, options.AgeKeyFile)
	if err != nil {
		return errors.Wrap(err, "failed to configure age key file")
	}

	err = ConfigureSshKeyFile(distributionName, options.SshKeyFile)
	if err != nil {
		return errors.Wrap(err, "failed to configure ssh key file")
	}
	err = ConfigureKustomizeUrl(distributionName, options.KustomizeUrl)
	if err != nil {
		return errors.Wrap(err, "failed to configure kustomize url")
	}

	err = RouteToWSL(distributionName, options.PersistentIPAddress)
	if err != nil {
		return errors.Wrap(err, "failed to add route")
	}

	if len(options.DomainNames) > 0 {
		_, err = ConfigureDomains(distributionName, options.PersistentIPAddress, options.DomainNames, false)
		if err != nil {
			return errors.Wrapf(err, "while configuring domains %s", strings.Join(options.DomainNames, ""))
		}
	}

	return err
}

func ConfigureDomains(distributionName, ipAddress string, domains []string, remove bool) ([]string, error) {
	if IsAdmin() {
		hosts, err := txeh.NewHostsDefault()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get hosts file")
		}
		if len(domains) == 0 {
			domains = hosts.ListHostsByIP(ipAddress)
			if !remove {
				return domains, nil
			}
		}
		if remove {
			hosts.RemoveHosts(domains)
		} else {
			hosts.AddHosts(ipAddress, domains)
		}
		err = hosts.Save()
		if err != nil {
			return nil, errors.Wrap(err, "failed to save hosts file")
		}
	} else {
		ctx := context.Background()
		client, err := GetElevatedClient()
		if err != nil {
			return nil, errors.Wrap(err, "while getting client")
		}
		response, err := client.ConfigureDomains(ctx, &ConfigureDomainsRequest{
			DistributionName: distributionName,
			IpAddress:        ipAddress,
			Domains:          domains,
			Remove:           remove,
		})
		if err == nil {
			domains = response.Domains
		} else {
			return nil, err
		}

	}
	if len(domains) == 0 {
		if remove {
			log.WithField("hosts", domains).Info("Removed hosts")
		} else {
			log.WithField("hosts", domains).Info("Current hosts")
		}

	} else {
		log.WithFields(log.Fields{
			"hosts":      domains,
			"ip_address": ipAddress,
		}).Info("Updated hosts")
	}

	return domains, nil
}
