package config

import (
	"github.com/spf13/viper"
)

type ConfigServer struct {
	UDPTargetAddr string `mapstructure:"udp_target_addr"`
	UDPListenAddr string `mapstructure:"udp_listen_addr"`
	WSListenAddr  string `mapstructure:"ws_listen_addr"`
	ListenPath    string `mapstructure:"listen_path"`
}

type ConfigClient struct {
	WSURL         string `mapstructure:"ws_url"`
	UDPListenAddr string `mapstructure:"udp_listen_addr"`
	UDPTargetAddr string `mapstructure:"udp_target_addr"`
	NMux          int    `mapstructure:"n_mux"`
}

type Config struct {
	Mode   string        `mapstructure:"mode"`
	Client *ConfigClient `mapstructure:"client"`
	Server *ConfigServer `mapstructure:"server"`
}

func New() *Config {
	return &Config{}
}

func (cfg *Config) Load() {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic("failed to read config file, error: " + err.Error())
	}
	if err := viper.UnmarshalKey("wsudp", cfg); err != nil {
		panic("failed to init api config, error: " + err.Error())
	}
}
