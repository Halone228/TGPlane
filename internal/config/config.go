package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	TDLib    TDLibConfig
	GRPC     GRPCConfig
	HTTP     HTTPConfig
	Log      LogConfig
	Auth      AuthConfig
	RateLimit RateLimitConfig
}

type AuthConfig struct {
	// MasterKey bypasses DB lookup. Leave empty to disable.
	MasterKey string `mapstructure:"master_key"`
}

type RateLimitConfig struct {
	// RPS is the sustained requests-per-second per key. 0 = disabled.
	RPS   float64 `mapstructure:"rps"`
	Burst int     `mapstructure:"burst"`
}

type AppConfig struct {
	Mode string `mapstructure:"mode"` // "main" | "worker"
	Name string `mapstructure:"name"`
}

type DatabaseConfig struct {
	DSN             string `mapstructure:"dsn"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime_seconds"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type TDLibConfig struct {
	APIID   int32  `mapstructure:"api_id"`
	APIHash string `mapstructure:"api_hash"`
	// Root directory where per-session data is stored: {DataDir}/{session_id}/
	DataDir   string `mapstructure:"data_dir"`
	LogLevel  int32  `mapstructure:"log_level"`
	UseTestDC bool   `mapstructure:"use_test_dc"`
}

type GRPCConfig struct {
	// For worker: address to connect to main node
	MainAddr string `mapstructure:"main_addr"`
	// For main: address to listen on
	ListenAddr string `mapstructure:"listen_addr"`
}

type HTTPConfig struct {
	Addr string `mapstructure:"addr"`
}

type LogConfig struct {
	Level string `mapstructure:"level"` // debug | info | warn | error
	JSON  bool   `mapstructure:"json"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.AutomaticEnv()

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.mode", "main")
	v.SetDefault("app.name", "tgplane")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.conn_max_lifetime_seconds", 300)
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("tdlib.data_dir", "./data/sessions")
	v.SetDefault("tdlib.log_level", 1)
	v.SetDefault("grpc.listen_addr", ":50051")
	v.SetDefault("http.addr", ":8080")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.json", false)
	v.SetDefault("rate_limit.rps", 100)
	v.SetDefault("rate_limit.burst", 200)
}
