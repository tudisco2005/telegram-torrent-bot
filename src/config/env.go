package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// EnvConfig holds environment configuration values
type EnvConfig struct {
	BotToken                *string
	Masters                 *MasterSlice
	RPCURL                  *string
	Username                *string
	Password                *string
	LogFile                 *string
	DefaultTorrentLocation  *string // directory where received .torrent files are saved before adding to Transmission
	DefaultDownloadLocation *string // directory where downloaded data will be stored
	DefaultMoveLocation     *string // directory where completed downloads should be copied/moved to
	NoLive                  *bool
	Verbose                 *bool
	UpdateMaxIterations     *int // max live-update iterations per message (0 = use Duration, no extra limit)
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

	// LogFile: check BOT_LOGFILE
	if cfg.LogFile != nil && *cfg.LogFile == "" {
		if logFile := os.Getenv("BOT_LOGFILE"); logFile != "" {
			*cfg.LogFile = logFile
		}
	}

	// DefaultTorrentLocation: check DEFAULT_TORRENT_LOCATION
	if cfg.DefaultTorrentLocation != nil && *cfg.DefaultTorrentLocation == "" {
		if dir := os.Getenv("DEFAULT_TORRENT_LOCATION"); dir != "" {
			*cfg.DefaultTorrentLocation = dir
		}
	}

	// DefaultDownloadLocation: check DEFAULT_DOWNLOAD_LOCATION
	if cfg.DefaultDownloadLocation != nil && *cfg.DefaultDownloadLocation == "" {
		// Primary: support legacy DEFAULT_DOWNLOAD_LOCATION env var
		if dir := os.Getenv("DEFAULT_DOWNLOAD_LOCATION"); dir != "" {
			*cfg.DefaultDownloadLocation = dir
		}
		// Secondary: support Transmission-specific env var name used in .env
		if dir := os.Getenv("TRANSMISSION_DONWNLOAD_LOCATION"); dir != "" && *cfg.DefaultDownloadLocation == "" {
			*cfg.DefaultDownloadLocation = dir
		}
	}

	// DefaultMoveLocation: check DEFAULT_MOVE_LOCATION
	if cfg.DefaultMoveLocation != nil && *cfg.DefaultMoveLocation == "" {
		if dir := os.Getenv("DEFAULT_MOVE_LOCATION"); dir != "" {
			*cfg.DefaultMoveLocation = dir
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

	// UpdateMaxIterations: check UPDATE_MAX_ITERATIONS (max live-update edits per message; 0 = disable live updates)
	if cfg.UpdateMaxIterations != nil {
		if v := os.Getenv("UPDATE_MAX_ITERATIONS"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				*cfg.UpdateMaxIterations = n
			}
		}
	}

	// Validate required parameters: panic if not set
	if cfg.BotToken != nil && *cfg.BotToken == "" {
		panic("config: required parameter TOKEN (or TT_BOTT) is not set")
	}
	if cfg.Masters != nil && len(*cfg.Masters) == 0 {
		panic("config: required parameter MASTER is not set")
	}
	if cfg.UpdateMaxIterations == nil || *cfg.UpdateMaxIterations == 0 {
		panic("config: required parameter UPDATE_MAX_ITERATIONS is not set")
	}
	if cfg.UpdateMaxIterations != nil && *cfg.UpdateMaxIterations < 0 {
		panic("config: UPDATE_MAX_ITERATIONS must be greater than or equal to 0")
	}
	if cfg.Username == nil || *cfg.Username == "" {
		panic("config: required parameter USERNAME is not set")
	}
	if cfg.Password == nil || *cfg.Password == "" {
		panic("config: required parameter PASSWORD is not set")
	}
	if cfg.RPCURL == nil || *cfg.RPCURL == "" {
		panic("config: required parameter RPC_URL is not set")
	}
}
