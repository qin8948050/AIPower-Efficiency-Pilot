package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Redis      RedisConfig      `mapstructure:"redis"`
	MySQL      MySQLConfig      `mapstructure:"mysql"`
	Prometheus PrometheusConfig `mapstructure:"prometheus"`
	K8s        K8sConfig        `mapstructure:"k8s"`
	LLM        LLMConfig        `mapstructure:"llm"`
}

type ServerConfig struct {
	Addr string `mapstructure:"addr"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type MySQLConfig struct {
	DSN string `mapstructure:"dsn"`
}

type PrometheusConfig struct {
	URL string `mapstructure:"url"`
}

type K8sConfig struct {
	Kubeconfig string `mapstructure:"kubeconfig"`
}

type LLMConfig struct {
	Provider   string `mapstructure:"provider"`   // "gemini", "openai", "minimax"
	APIKey     string `mapstructure:"api_key"`
	Model      string `mapstructure:"model"`       // "gemini-pro", "gpt-4", "MiniMax-Text-01"
	Endpoint   string `mapstructure:"endpoint"`    // 自定义端点
	MaxTokens  int    `mapstructure:"max_tokens"`  // 最大 token 数
	Temperature float64 `mapstructure:"temperature"` // 温度参数
}

func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// 支持环境变量覆盖 (例如: APP_MYSQL_DSN)
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %v", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &cfg, nil
}
