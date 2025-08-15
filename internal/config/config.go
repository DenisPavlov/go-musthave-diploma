package config

import (
	"log"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	Env string

	RunAddress string
	Server     struct {
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
		IdleTimeout  time.Duration
	}

	AccrualSystemAddress string
	DatabaseURI          string

	HTTPClient struct {
		Timeout  time.Duration
		RetryMax int
	}
}

func MustLoad() *Config {
	v := viper.New()

	setDefaults(v)
	readEnvVariables(v)
	readCommandLineFlags(v)

	var cfg Config
	err := v.Unmarshal(&cfg)
	if err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}

	return &cfg
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("env", "dev")
	v.SetDefault("runAddress", "localhost:8080")
	v.SetDefault("server.readTimeout", 5*time.Second)
	v.SetDefault("server.writeTimeout", 5*time.Second)
	v.SetDefault("server.idleTimeout", 60*time.Second)
	v.SetDefault("databaseUri", "postgresql://postgres:postgres@localhost:55432/gophermart?sslmode=disable")
	v.SetDefault("accrualSystemAddress", "http://localhost:8081")
	v.SetDefault("httpClient.timeout", 5*time.Second)
	v.SetDefault("httpClient.retryMax", 3)
}

func readEnvVariables(v *viper.Viper) {
	_ = v.BindEnv("env", "ENV")
	_ = v.BindEnv("runAddress", "RUN_ADDRESS")
	_ = v.BindEnv("databaseUri", "DATABASE_URI")
	_ = v.BindEnv("accrualSystemAddress", "ACCRUAL_SYSTEM_ADDRESS")
}

func readCommandLineFlags(v *viper.Viper) {
	pflag.String("env", "prod", "environment")
	pflag.String("a", "", "run address")
	pflag.String("d", "", "database uri")
	pflag.String("r", "", "accrual system address")
	pflag.Parse()

	_ = v.BindPFlag("env", pflag.Lookup("env"))
	_ = v.BindPFlag("runAddress", pflag.Lookup("a"))
	_ = v.BindPFlag("databaseUri", pflag.Lookup("d"))
	_ = v.BindPFlag("accrualSystemAddress", pflag.Lookup("r"))
}
