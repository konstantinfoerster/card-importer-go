package config

import (
	"fmt"
	"math"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Storage  Storage  `yaml:"storage"`
	Logging  Logging  `yaml:"logging"`
	Mtgjson  Mtgjson  `yaml:"mtgjson"`
	Scryfall Scryfall `yaml:"scryfall"`
	Database Database `yaml:"database"`
	HTTP     HTTP     `yaml:"http"`
}

type Database struct {
	Host           string `yaml:"host"`
	Port           string `yaml:"port"`
	Database       string `yaml:"database"`
	Username       string `yaml:"username"`
	Password       string `yaml:"password"`
	MaxConnections int32  `yaml:"maxConnections"`
}

func (d Database) ConnectionURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s/%s", d.Username, d.Password, net.JoinHostPort(d.Host, d.Port), d.Database)
}

func (d Database) MaxConnectionsOrDefault() int32 {
	if d.MaxConnections == 0 {
		defaultSize := int32(4)
		numCPU := runtime.NumCPU()
		if numCPU <= 0 || numCPU > math.MaxInt32 {
			panic("unsupported cpu count > maxInt32 or cpu count <= 0")
		}
		// #nosec G115 false positiv. This bug is fixed in latest gosec but not in golangci
		nCPU := int32(numCPU)
		if nCPU > defaultSize {
			return nCPU
		}

		return defaultSize
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
