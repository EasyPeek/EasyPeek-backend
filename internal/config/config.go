package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	CORS       CORSConfig       `mapstructure:"cors"`
	OpenRouter OpenRouterConfig `mapstructure:"open_router"`
}

type OpenRouterConfig struct {
	APIKey  string `yaml:"api_key"`
	APIHost string `yaml:"api_host"`
	Model   string `yaml:"model"`
}

var AppConfig *Config

func LoadConfig(filepath string) (*Config, error) {
	viper.SetConfigFile(filepath)
	viper.SetEnvPrefix("EasyPeek")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	AppConfig = &cfg

	return &cfg, nil
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"db_name"`
	SSLMode      string `mapstructure:"ssl_mode"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

// Data Source Name (DSN) for the database connection
// Example: "host=localhost user=gorm password=gorm dbname=gorm port=5432 sslmode=disable TimeZone=Asia/Shanghai"
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=Asia/Shanghai",
		d.Host, d.User, d.Password, d.DBName, d.Port, d.SSLMode)
}

type RedisConfig struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	Database int    `mapstructure:"database"`
}

type JWTConfig struct {
	SecretKey   string `mapstructure:"secret_key"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
}
