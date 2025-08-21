package config

import (
	"errors"
	"github.com/spf13/viper"
	"time"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"mysql"`
	Redis     RedisConfig     `yaml:"redis"`
	QQ        QQConfig        `yaml:"qq"`
	AliyunOss AliyunOssConfig `yaml:"aliyunoss"`
}

type ServerConfig struct {
	Port        int    `yaml:"port"`
	Env         string `yaml:"env"`
	StoragePath string `yaml:"storagePath"`
}

type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	DBName          string        `yaml:"dbname"`
	MaxOpenConns    int           `yaml:"maxOpenConns"`
	MaxIdleConns    int           `yaml:"maxIdleConns"`
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime"`
	ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime"`
	LogLevel        string        `yaml:"logLevel"`
}

type RedisConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"poolSize"`
}

type QQConfig struct {
	AppID       string `yaml:"appID"`       // QQ应用ID
	RedirectURI string `yaml:"redirectURI"` // 回调地址
	AppKey      string `yaml:"appKey"`      // QQ应用密钥
}

type AliyunOssConfig struct {
	Host         string `yaml:"host"`
	Bucket       string `yaml:"bucket"`
	EndPoint     string `yaml:"endPoint"`
	AccessId     string `yaml:"accessId"`
	AccessSecret string `yaml:"accessSecret"`
	Region       string `yaml:"region"`
}

func LoadConfig() (*Config, error) {
	v := viper.New()

	// 设置配置文件类型和路径
	v.SetConfigType("yaml")
	configPath := "conf/go-cloud-storage.dev.yaml"
	v.SetConfigFile(configPath)

	// 支持环境变量覆盖
	v.AutomaticEnv()

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	// 解析配置到结构体
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// 配置验证
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validateConfig(cfg *Config) error {
	if cfg.Database.Host == "" {
		return errors.New("mysql host is required")
	}
	if cfg.Database.DBName == "" {
		return errors.New("mysql name is required")
	}
	if cfg.Server.StoragePath == "" {
		return errors.New("server storage path is required")
	}
	return nil
}
