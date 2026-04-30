package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Profile struct {
	APIURL       string `yaml:"api-url" mapstructure:"api-url"`
	AuthURL      string `yaml:"auth-url" mapstructure:"auth-url"`
	ClientID     string `yaml:"client-id" mapstructure:"client-id"`
	ClientSecret string `yaml:"client-secret,omitempty" mapstructure:"client-secret"`
}

type Config struct {
	CurrentProfile string             `yaml:"current-profile"`
	Profiles       map[string]Profile `yaml:"profiles"`
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".beanctl/config.yaml"
	}
	return filepath.Join(home, ".beanctl", "config.yaml")
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultPath()
	}
	cfg := defaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.CurrentProfile == "" {
		cfg.CurrentProfile = "default"
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return cfg, nil
}

func Save(path string, cfg *Config) error {
	if path == "" {
		path = DefaultPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func (c *Config) Profile(name string) (Profile, bool) {
	if name == "" {
		name = c.CurrentProfile
	}
	p, ok := c.Profiles[name]
	return p, ok
}

func (c *Config) SetProfile(name string, p Profile) {
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	c.Profiles[name] = p
	if c.CurrentProfile == "" {
		c.CurrentProfile = name
	}
}

func defaultConfig() *Config {
	return &Config{
		CurrentProfile: "default",
		Profiles: map[string]Profile{
			"default": {
				APIURL:   "https://beancs.hollowdata.com",
				AuthURL:  "https://auth.beancs.hollowdata.com",
				ClientID: "beanctl-cli",
			},
			"local": {
				APIURL:   "http://localhost:8080",
				AuthURL:  "http://localhost:8101/api/v1",
				ClientID: "beanctl-local",
			},
		},
	}
}
