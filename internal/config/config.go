package config

import (
	"fmt"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"

	"fuck-watermark/logs"
)

type Config struct {
	Addr           int64
	RequestTimeout time.Duration
	WeiboProxyBase string
	DouyinCookie   string
	Security       SecurityConfig
	RateLimit      RateLimitConfig
	Log            logs.Config
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

type ServerConfig struct {
	Addr           int64
	RequestTimeout time.Duration
}

type Weibo struct {
	ProxyBase string `toml:"proxy_base"`
}

type Douyin struct {
	Cookie string `toml:"cookie"`
}

type Security struct {
	APIKey       string   `toml:"api_key"`
	AllowOrigins []string `toml:"allow_origins"`
	MaxBodyBytes int64    `toml:"max_body_bytes"`
}

type RateLimit struct {
	Enabled           bool `toml:"enabled"`
	RequestsPerMinute int  `toml:"requests_per_minute"`
	Burst             int  `toml:"burst"`
}

type fileConfig struct {
	Server    ServerConfig
	Weibo     Weibo
	Douyin    Douyin
	Security  Security
	RateLimit RateLimit
	Log       logs.Config `toml:"log"`
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

	logCfg := fc.Log
	defLog := logs.DefaultConfig()
	if logCfg.Level == "" {
		logCfg.Level = defLog.Level
	}
	if logCfg.Format == "" {
		logCfg.Format = defLog.Format
	}
	if logCfg.LogPath == "" {
		logCfg.LogPath = defLog.LogPath
	}
	if logCfg.Filename == "" {
		logCfg.Filename = defLog.Filename
	}
	if logCfg.MaxSize <= 0 {
		logCfg.MaxSize = defLog.MaxSize
	}
	if logCfg.MaxBackups <= 0 {
		logCfg.MaxBackups = defLog.MaxBackups
	}
	if logCfg.MaxAge <= 0 {
		logCfg.MaxAge = defLog.MaxAge
	}
	// toml 未写 console 时为零值 false；无完整 log 段时用默认 true
	if !logCfg.Console && fc.Log.Level == "" && fc.Log.Format == "" && fc.Log.LogPath == "" {
		logCfg.Console = defLog.Console
		logCfg.Compress = defLog.Compress
	}

	return Config{
		Addr:           addr,
		RequestTimeout: timeout,
		WeiboProxyBase: fc.Weibo.ProxyBase,
		DouyinCookie:   fc.Douyin.Cookie,
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
		Log: logCfg,
	}, nil
}
