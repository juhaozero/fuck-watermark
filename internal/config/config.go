package config

import (
	"fmt"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Addr           int64
	RequestTimeout time.Duration
	WeiboProxyBase string
	Security       SecurityConfig
	RateLimit      RateLimitConfig
}

type SecurityConfig struct {
	APIKey       string
	AllowOrigins []string
	MaxBodyBytes int64
}

type RateLimitConfig struct {
	Enabled           bool
	RequestsPerMinute int
	Burst             int
}

type fileConfig struct {
	Server struct {
		Addr           int64 `toml:"addr"`
		RequestTimeout int   `toml:"request_timeout"`
	} `toml:"server"`
	Weibo struct {
		ProxyBase string `toml:"proxy_base"`
	} `toml:"weibo"`
	Security struct {
		APIKey       string   `toml:"api_key"`
		AllowOrigins []string `toml:"allow_origins"`
		MaxBodyBytes int64    `toml:"max_body_bytes"`
	} `toml:"security"`
	RateLimit struct {
		Enabled           bool `toml:"enabled"`
		RequestsPerMinute int  `toml:"requests_per_minute"`
		Burst             int  `toml:"burst"`
	} `toml:"rate_limit"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("读取配置文件 %q: %w", path, err)
	}

	var fc fileConfig
	if err := toml.Unmarshal(data, &fc); err != nil {
		return Config{}, fmt.Errorf("解析配置文件 %q: %w", path, err)
	}

	timeout := 15 * time.Second
	if fc.Server.RequestTimeout > 0 {
		timeout = time.Duration(fc.Server.RequestTimeout) * time.Second
	}

	addr := fc.Server.Addr
	if addr == 0 {
		addr = 8080
	}

	maxBody := fc.Security.MaxBodyBytes
	if maxBody <= 0 {
		maxBody = 16384
	}

	origins := fc.Security.AllowOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}

	rpm := fc.RateLimit.RequestsPerMinute
	if rpm <= 0 {
		rpm = 60
	}
	burst := fc.RateLimit.Burst
	if burst <= 0 {
		burst = 10
	}

	return Config{
		Addr:           addr,
		RequestTimeout: timeout,
		WeiboProxyBase: fc.Weibo.ProxyBase,
		Security: SecurityConfig{
			APIKey:       fc.Security.APIKey,
			AllowOrigins: origins,
			MaxBodyBytes: maxBody,
		},
		RateLimit: RateLimitConfig{
			Enabled:           fc.RateLimit.Enabled,
			RequestsPerMinute: rpm,
			Burst:             burst,
		},
	}, nil
}
