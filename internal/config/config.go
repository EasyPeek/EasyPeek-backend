package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	CORS     CORSConfig     `mapstructure:"cors"`
	AI       AIConfig       `mapstructure:"ai"`
}

var AppConfig *Config

// LoadConfig 加载并解析配置文件
func LoadConfig(filepath string) (*Config, error) {
	// 指定配置文件
	viper.SetConfigFile(filepath)

	// 从文件读取配置
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	// 解码到结构体
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

// DSN 构造函数
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=Asia/Shanghai",
		d.Host, d.User, d.Password, d.DBName, d.Port, d.SSLMode,
	)
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
	Provider     string             `mapstructure:"provider"`
	APIKey       string             `mapstructure:"api_key"`
	BaseURL      string             `mapstructure:"base_url"`
	Model        string             `mapstructure:"model"`
	Timeout      int                `mapstructure:"timeout"`
	MaxTokens    int                `mapstructure:"max_tokens"`
	Temperature  float64            `mapstructure:"temperature"`
	SiteURL      string             `mapstructure:"site_url"`
	SiteName     string             `mapstructure:"site_name"`
	AutoAnalysis AutoAnalysisConfig `mapstructure:"auto_analysis"`
}

type AutoAnalysisConfig struct {
	Enabled              bool `mapstructure:"enabled"`
	AnalyzeOnFetch       bool `mapstructure:"analyze_on_fetch"`
	BatchProcessInterval int  `mapstructure:"batch_process_interval"`
	MaxBatchSize         int  `mapstructure:"max_batch_size"`
	AnalysisDelay        int  `mapstructure:"analysis_delay"`
}
