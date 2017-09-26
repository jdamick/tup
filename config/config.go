package config

import (
	"time"

	"github.com/Sirupsen/logrus"
)

type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

var (
	DefaultLogger = logrus.StandardLogger()
)

type Config struct {
	Logger    Logger        // otherwise DefaultLogger is used
	ProxyAddr string        `json:"addr"` // host:port
	Backends  []Backend     `json:"backends"`
	Timeout   time.Duration `json:"timeout"`
	Debug     bool          `json:"debug"`
}

func DefaultConfig() *Config {
	return &Config{Logger: DefaultLogger}
}

type Backend struct {
	Addr string `json:"addr"` // host:port
}

func (c *Config) Log() Logger {
	if c.Logger == nil {
		return DefaultLogger
	}
	return c.Logger
}
