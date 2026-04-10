package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds runtime settings for LAN Mapper.
type Config struct {
	HTTPPort        int           `mapstructure:"http_port"`
	DataDir         string        `mapstructure:"data_dir"`
	ScanCIDR        []string      `mapstructure:"scan_cidr"`
	SNMPCommunities []string      `mapstructure:"snmp_communities"`
	ScanInterval    time.Duration `mapstructure:"scan_interval"`
	AdminToken      string        `mapstructure:"admin_token"`
}

// Load reads configuration from config files and environment variables.
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("/app/data")
	v.AutomaticEnv()
	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}
