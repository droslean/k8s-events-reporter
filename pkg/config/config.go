package config

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/ghodss/yaml"
)

// Config holds the configuration details.
type Config struct {
	EmailSettings EmailSettings     `json:"email_settings,omitempty"`
	Reports       map[string]Report `json:"reports,omitempty"`
}

// EmailSettings holds information and authentication details for the smtp server.
type EmailSettings struct {
	SMTPServer string `json:"smtp_server,omitempty"`
	Port       int    `json:"port,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
}

// Report holds the reports information.
type Report struct {
	Description     string   `json:"description,omitempty"`
	Kind            string   `json:"kind,omitempty"`
	Reasons         []string `json:"reasons,omitempty"`
	IntervalString  string   `json:"interval,omitempty"`
	EmailRecipients []string `json:"email_recipients,omitempty"`
	Interval        time.Duration
}

// NewConfig creates new config.
func NewConfig(file string) (*Config, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not read the config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	if err := config.parseConfig(); err != nil {
		return nil, fmt.Errorf("could not parse config: %v", err)
	}
	return &config, nil
}

func (c *Config) parseConfig() error {
	for name, report := range c.Reports {
		if report.IntervalString == "" {
			report.Interval = 24 * time.Hour
		} else {
			interval, err := time.ParseDuration(report.IntervalString)
			if err != nil {
				return fmt.Errorf("cannot parse duration for interval: %v", err)
			}
			report.Interval = interval
		}
		c.Reports[name] = report
	}
	return nil
}
