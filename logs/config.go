package logs

// Config 日志配置。
type Config struct {
	// Level 日志级别: debug / info / warn / error / fatal / panic
	Level string `toml:"level" koanf:"level"`
	// Format 输出格式: json / console
	Format string `toml:"format" koanf:"format"`
	// LogPath 日志文件路径
	LogPath string `toml:"log_path" koanf:"log_path"`
	// Filename 日志文件名称
	Filename string `toml:"filename" koanf:"filename"`
	// MaxSize 单个日志文件最大体积（MB），超出后自动切分
	MaxSize int `toml:"max_size" koanf:"max_size"`
	// MaxBackups 保留的历史日志文件数量
	MaxBackups int `toml:"max_backups" koanf:"max_backups"`
	// MaxAge 日志文件最大保留天数
	MaxAge int `toml:"max_age" koanf:"max_age"`
	// Compress 是否压缩已切分的旧日志
	Compress bool `toml:"compress" koanf:"compress"`
	// Console 是否同时输出到控制台
	Console bool `toml:"console" koanf:"console"`
	// IsDebug 是否为调试模式
	IsDebug bool `toml:"is_debug" koanf:"is_debug"`
}

// DefaultConfig 返回默认配置。
func DefaultConfig() Config {
	return Config{
		Level:      "info",
		Format:     "json",
		LogPath:    "var/logs",
		Filename:   "app",
		MaxSize:    100,
		MaxBackups: 10,
		MaxAge:     30,
		Compress:   true,
		Console:    true,
	}
}
