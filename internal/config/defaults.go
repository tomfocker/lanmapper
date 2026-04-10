package config

import "github.com/spf13/viper"

func setDefaults(v *viper.Viper) {
	v.SetDefault("http_port", 8080)
	v.SetDefault("data_dir", "./data")
	v.SetDefault("scan_cidr", []string{})
	v.SetDefault("snmp_communities", []string{"public", "private"})
	v.SetDefault("scan_interval", "0s")
}
