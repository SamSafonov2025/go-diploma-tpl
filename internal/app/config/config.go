package config

import (
	"flag"
	"sync"

	"github.com/caarlos0/env/v6"
	"go.uber.org/zap"
)

type Configuration struct {
	ServerAddr       string `env:"RUN_ADDRESS" envDefault:":8080"`
	AccrualSystemURL string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	DatabaseDSN      string `env:"DATABASE_URI"`
	TokenSecret      string `env:"TOKEN_SIGN_KEY" envDefault:"The Little Man Who Wasn't There"`
}

var (
	cfg  *Configuration
	once sync.Once
)

func Load() *Configuration {
	once.Do(func() {
		cfg = &Configuration{}

		// Parse environment variables
		if err := env.Parse(cfg); err != nil {
			zap.L().Fatal("Failed to parse environment variables", zap.Error(err))
		}

		// Parse command-line flags
		flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "server address and port")
		flag.StringVar(&cfg.AccrualSystemURL, "r", cfg.AccrualSystemURL, "accrual system URL")
		flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "database connection string")
		flag.Parse()
	})
	return cfg
}
