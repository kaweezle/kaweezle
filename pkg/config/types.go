package config

type ConfigurationOptions struct {
	PersistentIPAddress string
	AgeKeyFile          string
	SshKeyFile          string
	KustomizeUrl        string
	DomainNames         []string
	SshHosts            []string
}

func NewConfigurationOptions() *ConfigurationOptions {
	options := &ConfigurationOptions{}
	options.PersistentIPAddress = "192.168.99.2"
	options.SshHosts = []string{"github.com", "gitlab.com"}
	return options
}
