package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// EnvConfig holds environment configuration values
type EnvConfig struct {
	BotToken     *string
	Masters      *MasterSlice
	RPCURL       *string
	Username     *string
	Password     *string
	LogFile      *string
	TransLogFile *string
	NoLive       *bool
	Verbose      *bool
}

// LoadEnvironmentConfig loads configuration from .env file and environment variables
// It only sets values that are not already set (empty strings, false booleans, etc.)
func LoadEnvironmentConfig(cfg *EnvConfig) {
	// Load .env file if it exists (ignore errors if file doesn't exist)
	// Try loading from current directory first, then from parent directory
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	// BotToken: check TT_BOTT, TOKEN
	if cfg.BotToken != nil && *cfg.BotToken == "" {
		if token := os.Getenv("TT_BOTT"); token != "" {
			*cfg.BotToken = token
		} else if token := os.Getenv("TOKEN"); token != "" {
			*cfg.BotToken = token
		}
	}

	// Masters: check MASTER
	if cfg.Masters != nil && len(*cfg.Masters) == 0 {
		if masterEnv := os.Getenv("MASTER"); masterEnv != "" {
			// Support comma-separated masters
			masters := strings.Split(masterEnv, ",")
			for _, m := range masters {
				m = strings.TrimSpace(m)
				if m != "" {
					// Remove @ if present and convert to lowercase
					m = strings.Replace(m, "@", "", -1)
					m = strings.ToLower(m)
					*cfg.Masters = append(*cfg.Masters, m)
				}
			}
		}
	}

	// Username: check TR_AUTH, USERNAME
	if cfg.Username != nil && *cfg.Username == "" {
		if username := os.Getenv("TR_AUTH"); username != "" {
			*cfg.Username = username
		} else if username := os.Getenv("USERNAME"); username != "" {
			*cfg.Username = username
		}
	}

	// Password: check PASSWORD
	if cfg.Password != nil && *cfg.Password == "" {
		if password := os.Getenv("PASSWORD"); password != "" {
			*cfg.Password = password
		}
	}

	// RPCURL: check RPC_URL (only if default value)
	if cfg.RPCURL != nil && *cfg.RPCURL == "http://localhost:9091/transmission/rpc" {
		if urlEnv := os.Getenv("RPC_URL"); urlEnv != "" {
			*cfg.RPCURL = urlEnv
		}
	}

	// LogFile: check LOGFILE
	if cfg.LogFile != nil && *cfg.LogFile == "" {
		if logFile := os.Getenv("LOGFILE"); logFile != "" {
			*cfg.LogFile = logFile
		}
	}

	// TransLogFile: check TRANSMISSION_LOGFILE
	if cfg.TransLogFile != nil && *cfg.TransLogFile == "" {
		if transLogFile := os.Getenv("TRANSMISSION_LOGFILE"); transLogFile != "" {
			*cfg.TransLogFile = transLogFile
		}
	}

	// NoLive: check NO_LIVE
	if cfg.NoLive != nil && !*cfg.NoLive {
		if noLiveEnv := os.Getenv("NO_LIVE"); noLiveEnv != "" {
			if noLive, err := strconv.ParseBool(noLiveEnv); err == nil {
				*cfg.NoLive = noLive
			}
		}
	}

	// Verbose: check VERBOSE (if "1" or "true", set to true)
	if cfg.Verbose != nil && !*cfg.Verbose {
		if verboseEnv := os.Getenv("VERBOSE"); verboseEnv != "" {
			verboseEnv = strings.ToLower(strings.TrimSpace(verboseEnv))
			if verboseEnv == "1" || verboseEnv == "true" {
				*cfg.Verbose = true
			}
		}
	}
}
