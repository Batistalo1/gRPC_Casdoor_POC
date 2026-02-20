package config

import (
	"fmt"

	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfigdotenv"
)

type Config struct {
	ClientID     string `env:"CLIENT_ID"`
	ClientSecret string `env:"CLIENT_SECRET"`
	CasdoorURL   string `env:"CASDOOR_URL"`
	ServerURL    string `env:"SERVER_URL"`
	ServerPort   string `env:"SERVER_PORT"`
	CallbackURL  string `env:"CALLBACK_URL"`
	AuthURL      string `env:"AUTH_URL"`
	TokenURL     string `env:"TOKEN_URL"`
	DatabaseURL  string `env:"DATABASE_URL"`
	Organization string `env:"ORGANIZATION"`
	AppName      string `env:"APP_NAME"`
}

func NewConfig() (Config, error) {
	config := Config{}

	loader := aconfig.LoaderFor(&config, aconfig.Config{
		EnvPrefix: "",
		Files:     []string{".env"},
		FileDecoders: map[string]aconfig.FileDecoder{
			".env": aconfigdotenv.New(),
		},
	})

	err := loader.Load()
	if err != nil {
		return Config{}, fmt.Errorf("failed to load the configuration: %w", err)
	}

	return config, nil
}
