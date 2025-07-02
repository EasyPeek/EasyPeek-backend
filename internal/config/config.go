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
	AI         AIConfig         `mapstructure:"ai"`
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

type AIConfig struct {
	Provider    string  `mapstructure:"provider"`
	APIKey      string  `mapstructure:"api_key"`
	BaseURL     string  `mapstructure:"base_url"`
	Model       string  `mapstructure:"model"`
	Timeout     int     `mapstructure:"timeout"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	Temperature float64 `mapstructure:"temperature"`
	// OpenRouter特有配置
	SiteURL  string `mapstructure:"site_url"`  // 你的网站URL
	SiteName string `mapstructure:"site_name"` // 你的应用名称
	// 自动分析配置
	AutoAnalysis AutoAnalysisConfig `mapstructure:"auto_analysis"`
}

type AutoAnalysisConfig struct {
	Enabled              bool `mapstructure:"enabled"`                // 是否启用自动AI分析
	AnalyzeOnFetch       bool `mapstructure:"analyze_on_fetch"`       // 在RSS抓取时即时分析
	BatchProcessInterval int  `mapstructure:"batch_process_interval"` // 批处理间隔（分钟）
	MaxBatchSize         int  `mapstructure:"max_batch_size"`         // 每次批处理的最大数量
	AnalysisDelay        int  `mapstructure:"analysis_delay"`         // 每个分析之间的延迟（秒）
}
