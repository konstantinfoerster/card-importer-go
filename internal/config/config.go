package config

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Logging  Logging  `yaml:"logging"`
	Database Database `yaml:"database"`
	HTTP     HTTP     `yaml:"http"`
	Mtgjson  Mtgjson  `yaml:"mtgjson"`
	Scryfall Scryfall `yaml:"scryfall"`
	Storage  Storage  `yaml:"storage"`
}

type Database struct {
	Host           string `yaml:"host"`
	Port           string `yaml:"port"`
	Database       string `yaml:"database"`
	Username       string `yaml:"username"`
	Password       string `yaml:"password"`
	MaxConnections int    `yaml:"maxConnections"`
}

func (d Database) ConnectionURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s/%s", d.Username, d.Password, net.JoinHostPort(d.Host, d.Port), d.Database)
}

func (d Database) MaxConnectionsOrDefault() int {
	if d.MaxConnections == 0 {
		return runtime.NumCPU()
	}

	return d.MaxConnections
}

type HTTP struct {
	Timeout time.Duration `yaml:"timeout"`
}

type Mtgjson struct {
	DownloadURL string `yaml:"downloadUrl"`
}

type Logging struct {
	Level string `yaml:"level"`
}

func (l Logging) LevelOrDefault() string {
	level := strings.TrimSpace(l.Level)
	if level == "" {
		level = "INFO"
	}

	return strings.ToLower(level)
}

type Scryfall struct {
	DownloadURL string `yaml:"downloadUrl"`
}

func (i Scryfall) BuildJSONDownloadURL(setCode string, cardNumber string, lang string) string {
	r := strings.NewReplacer("{code}", setCode, "{number}", cardNumber, "{lang}", lang, "{format}", "json")

	return strings.ToLower(r.Replace(i.DownloadURL))
}

const (
	REPLACE = "REPLACE"
	CREATE  = "CREATE"
)

type Storage struct {
	Location string `yaml:"location"`
	Mode     string `yaml:"mode"`
}

func Load(path string) (*Config, error) {
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
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("can't read config file: %w", err)
	}

	config := &Config{}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("config unmarshal failed with: %w", err)
	}

	// TODO validate config content

	return config, nil
}
