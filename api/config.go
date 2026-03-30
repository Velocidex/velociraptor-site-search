package api

import (
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/goccy/go-yaml"
)

type DynDNS struct {
	Type          string `json:"type"`
	ApiToken      string `json:"api_token"`
	ZoneName      string `json:"zone_name"`
	ExternalIPURL string `json:"external_ip_url"`
	Frequency     uint64 `json:"frequency"`
}

type Config struct {
	IndexPath      string `json:"index_path""`
	IndexURL       string `json:"index_url"`
	MaxIndexAgeSec uint64 `json:"max_index_age_sec"`

	DynDns *DynDNS `json:"dyndns"`

	// When using autocert we always bind to 0.0.0.0:443
	BindAddress string `json:"bind_addr"`
	Hostname    string `json:"hostname"`

	// If this is empty we use plain http
	AutocertCertCache string `json:"autocert_cachedir"`
	LogToSyslog       bool   `json:"syslog"`

	// Set to true to emit debugging messages.
	Debug bool `json:"debug"`
}

func (self *Config) GetLogger() *Logger {
	return &Logger{}
}

func LoadFromFile(path string) (*Config, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(fd)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	return config, VerifyConfig(config)
}

func VerifyConfig(config *Config) (err error) {
	if config.LogToSyslog {
		syslog_logger, err = NewSyslogLogger()
		if err != nil {
			return err
		}
	}

	if !config.Debug {
		syslog_logger.debug_logger = log.New(io.Discard, "", 0)
	}

	return nil
}
