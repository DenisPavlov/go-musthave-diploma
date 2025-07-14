package config

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"log"
	"strings"
)

type Config struct {
	Env                  string `mapstructure:"env"`
	RunAddress           string `mapstructure:"run_address"`
	AccrualSystemAddress string `mapstructure:"accrual_system_address"`
}

func MustLoad() *Config {

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	err := initFlags()
	if err != nil {
		log.Fatalf("failed to initialize flags, %v", err)
	}

	var cfg Config

	err = viper.Unmarshal(&cfg)
	if err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}

	return &cfg
}

func initFlags() error {
	pflag.String("run_address", "", "system address")
	pflag.String("accrual_system_address", "", "accrual system address")
	pflag.String("env", "prod", "environment")

	pflag.Parse()
	return viper.BindPFlags(pflag.CommandLine)
}
