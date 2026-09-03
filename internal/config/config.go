package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	defaultListen       = "127.0.0.1:5232"
	defaultDataDir      = "./data"
	defaultMaxBodyBytes = int64(10 * 1024 * 1024)
)

type Config struct {
	Listen       string
	DataDir      string
	Username     string
	Password     string
	MaxBodyBytes int64
}

func Load() (Config, error) {
	cfg := Config{
		Listen:       envOr("GOREECLOUD_DAV_LISTEN", defaultListen),
		DataDir:      envOr("GOREECLOUD_DAV_DATA_DIR", defaultDataDir),
		Username:     os.Getenv("GOREECLOUD_DAV_USERNAME"),
		Password:     os.Getenv("GOREECLOUD_DAV_PASSWORD"),
		MaxBodyBytes: defaultMaxBodyBytes,
	}

	if raw := strings.TrimSpace(os.Getenv("GOREECLOUD_DAV_MAX_BODY_BYTES")); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("invalid GOREECLOUD_DAV_MAX_BODY_BYTES")
		}
		cfg.MaxBodyBytes = n
	}

	if (cfg.Username == "") != (cfg.Password == "") {
		return Config{}, fmt.Errorf("GOREECLOUD_DAV_USERNAME and GOREECLOUD_DAV_PASSWORD must be configured together")
	}
	if cfg.Username != "" && !validDevelopmentPrincipalID(cfg.Username) {
		return Config{}, fmt.Errorf("GOREECLOUD_DAV_USERNAME must be a canonical development principal ID using only letters, digits, '-' or '_'")
	}

	if !isLoopbackListen(cfg.Listen) && cfg.Username == "" {
		return Config{}, fmt.Errorf("refusing non-loopback listener without development credentials")
	}

	return cfg, nil
}

func envOr(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}

func isLoopbackListen(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validDevelopmentPrincipalID(value string) bool {
	if value == "" || value == "." || value == ".." || strings.HasPrefix(value, ".") {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}
