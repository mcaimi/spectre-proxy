package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Proxy    ProxyConfig
	WebUI    WebUIConfig
	Database DatabaseConfig
	TLS      TLSConfig
	Logging  LoggingConfig
}

type ProxyConfig struct {
	HTTPPort       int
	HTTPSPort      int
	BindAddr       string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	MaxConnections int
}

type WebUIConfig struct {
	Enabled  bool
	Port     int
	BindAddr string
}

type DatabaseConfig struct {
	Path         string
	MaxOpenConns int
	MaxIdleConns int
}

type TLSConfig struct {
	CertCacheSize int
	CAKeySize     int
	CertKeySize   int
	CertValidity  time.Duration
}

type LoggingConfig struct {
	Level  string
	Format string
	Output string
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	setDefaults(v)

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("spectre")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.spectre")
		v.AddConfigPath("/etc/spectre")
	}

	v.SetEnvPrefix("SPECTRE")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("proxy.httpport", 8080)
	v.SetDefault("proxy.httpsport", 8443)
	v.SetDefault("proxy.bindaddr", "0.0.0.0")
	v.SetDefault("proxy.readtimeout", 30*time.Second)
	v.SetDefault("proxy.writetimeout", 30*time.Second)
	v.SetDefault("proxy.idletimeout", 120*time.Second)
	v.SetDefault("proxy.maxconnections", 1000)

	v.SetDefault("webui.enabled", true)
	v.SetDefault("webui.port", 9000)
	v.SetDefault("webui.bindaddr", "127.0.0.1")

	v.SetDefault("database.path", "./spectre.db")
	v.SetDefault("database.maxopenconns", 25)
	v.SetDefault("database.maxidleconns", 5)

	v.SetDefault("tls.certcachesize", 100)
	v.SetDefault("tls.cakeysize", 4096)
	v.SetDefault("tls.certkeysize", 2048)
	v.SetDefault("tls.certvalidity", 365*24*time.Hour)

	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("logging.output", "stdout")
}
