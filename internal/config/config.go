package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

type Config struct {
	Logging  Logging  `yaml:"logging"`
	Database Database `yaml:"database"`
	Mtgjson  Mtgjson  `yaml:"mtgjson"`
}

type Database struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Mtgjson struct {
	DownloadURL string `yaml:"downloadURL"`
}

type Logging struct {
	Level string `yaml:"level"`
}

func (l *Logging) LevelOrDefault() string {
	level := strings.TrimSpace(l.Level)
	if level == "" {
		level = "INFO"
	}
	return strings.ToLower(level)
}

func (d *Database) ConnectionUrl() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", d.Username, d.Password, d.Host, d.Port, d.Database)
}

var doOnce sync.Once
var cfg *Config

func Load(path string) error {
	var err error
	doOnce.Do(func() {
		cfg, err = loadConfig(path)
	})

	return err
}

func loadConfig(path string) (*Config, error) {
	s, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if s.IsDir() {
		return nil, fmt.Errorf("'%s' is a directory, not a regular file", path)
	}

	return buildConfig(path)
}

func buildConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("can't read config file: %s", err.Error())
	}

	config := &Config{}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("config unmarshal failed with: %s", err.Error())
	}

	return config, nil
}

func Get() *Config {
	return cfg
}
