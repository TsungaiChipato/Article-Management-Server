package env

import "github.com/caarlos0/env/v10"

type config struct {
	MongodPath string `env:"MONGOD_PATH" envDefault:"/usr/local/bin/mongod"` // TODO: find solution for this for testing
}

func Load() (*config, error) {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
