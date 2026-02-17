package config

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// Config holds the application configuration
type Config struct {
	BotToken     string
	Masters      []string
	RPCURL       string
	Username     string
	Password     string
	LogFile      string
	NoLive       bool
	Logger       *log.Logger
}

// MasterSlice is a type for masters for the flag package to parse them as a slice
type MasterSlice []string

// String is mandatory function for the flag package
func (masters *MasterSlice) String() string {
	return fmt.Sprintf("%s", *masters)
}

// Set is mandatory function for the flag package
func (masters *MasterSlice) Set(master string) error {
	*masters = append(*masters, strings.ToLower(master))
	return nil
}

// Contains takes a string and return true if MasterSlice has it
func (masters MasterSlice) Contains(master string) bool {
	master = strings.ToLower(master)
	for i := range masters {
		if strings.ToLower(masters[i]) == master {
			return true
		}
	}
	return false
}

// NewConfig creates a new Config instance with default logger
func NewConfig() *Config {
	return &Config{
		Logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// SetupLogging configures logging to a file if specified
func (c *Config) SetupLogging() error {
	if c.LogFile != "" {
		logf, err := os.OpenFile(c.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		c.Logger.SetOutput(logf)
	}
	return nil
}

// LogConfiguration logs the current configuration
func (c *Config) LogConfiguration() {
	c.Logger.Printf("[INFO] Token=%s\n\t\tMasters=%s\n\t\tURL=%s\n\t\tUSER=%s\n\t\tPASS=%s",
		c.BotToken, c.Masters, c.RPCURL, c.Username, c.Password)
}
