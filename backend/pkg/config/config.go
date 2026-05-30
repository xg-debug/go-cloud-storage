package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server      ServerConfig    `yaml:"server"`
	Database    DatabaseConfig  `yaml:"mysql"`
	Redis       RedisConfig     `yaml:"redis"`
	RabbitMQ    RabbitMQConfig  `yaml:"rabbitmq" mapstructure:"rabbitmq"`
	QQ          QQConfig        `yaml:"qq"`
	AliyunOss   AliyunOssConfig `yaml:"aliyun"`
	StorageType string          `yaml:"storageType"`
	JWT         JWTConfig       `yaml:"jwt"`
	Minio       MinioConfig     `yaml:"minio"`
	Security    SecurityConfig  `yaml:"security"`
}

type SecurityConfig struct {
	AllowedExtensions []string `yaml:"allowedExtensions"` // 允许上传的文件扩展名，空表示全部允许
	MaxFileSizeMB     int      `yaml:"maxFileSizeMB"`     // 单文件最大大小(MB)，0表示不限制
	RateLimitRPS      int      `yaml:"rateLimitRPS"`      // 每用户每秒最大请求数，0表示不限制
	DefaultQuotaGB    int      `yaml:"defaultQuotaGB"`    // 新用户默认配额(GB)，默认10
}

type JWTConfig struct {
	Secret string `yaml:"secret"`
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

type RabbitMQConfig struct {
	Enabled             bool   `yaml:"enabled" mapstructure:"enabled"`
	URL                 string `yaml:"url" mapstructure:"url"`
	Exchange            string `yaml:"exchange" mapstructure:"exchange"`
	Queue               string `yaml:"queue" mapstructure:"queue"`
	RoutingKey          string `yaml:"routingKey" mapstructure:"routingKey"`
	ConsumerTag         string `yaml:"consumerTag" mapstructure:"consumerTag"`
	ScanIntervalSeconds int    `yaml:"scanIntervalSeconds" mapstructure:"scanIntervalSeconds"`
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

type MinioConfig struct {
	Endpoint        string `yaml:"endpoint"`
	AccessKeyID     string `yaml:"accessKeyID"`
	SecretAccessKey string `yaml:"secretAccessKey"`
	Bucket          string `yaml:"bucket"`
	UseSSL          bool   `yaml:"useSSL"`
	Region          string `yaml:"region"`
}

func LoadConfig() (*Config, error) {
	v := viper.New()

	// 设置配置文件名称（不带后缀）
	v.SetConfigName("go-cloud-storage.dev")
	// 设置配置文件类型
	v.SetConfigType("yaml")

	// 添加搜索路径：
	// 1. 当前目录下的 conf 文件夹
	v.AddConfigPath("conf")
	// 2. 上级目录下的 conf 文件夹（适配从 cmd 目录下运行的情况）
	v.AddConfigPath("../conf")
	// 3. backend 目录下的 conf 文件夹（适配从项目根目录运行的情况）
	v.AddConfigPath("backend/conf")
	// 4. 项目根目录
	v.AddConfigPath(".")

	// 支持环境变量覆盖（敏感信息优先从环境变量读取）
	v.AutomaticEnv()
	v.SetEnvPrefix("GCS") // GCS_DATABASE_PASSWORD 等
	v.BindEnv("database.password", "GCS_DB_PASSWORD")
	v.BindEnv("redis.password", "GCS_REDIS_PASSWORD")
	v.BindEnv("minio.accessKeyID", "GCS_MINIO_ACCESS_KEY")
	v.BindEnv("minio.secretAccessKey", "GCS_MINIO_SECRET_KEY")
	v.BindEnv("jwt.secret", "GCS_JWT_SECRET")

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
	// 安全配置默认值
	if cfg.Security.DefaultQuotaGB <= 0 {
		cfg.Security.DefaultQuotaGB = 10
	}
	if cfg.Security.MaxFileSizeMB <= 0 {
		cfg.Security.MaxFileSizeMB = 500 // 默认500MB
	}
	if cfg.Security.RateLimitRPS <= 0 {
		cfg.Security.RateLimitRPS = 50 // 默认每秒50次
	}
	if cfg.Security.AllowedExtensions == nil {
		// 默认允许常见文件类型
		cfg.Security.AllowedExtensions = []string{
			".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg",
			".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv",
			".mp3", ".wav", ".flac", ".aac", ".ogg",
			".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
			".txt", ".md", ".csv", ".json", ".xml", ".yaml", ".yml",
			".zip", ".rar", ".7z", ".tar", ".gz",
			".html", ".htm", ".css", ".js", ".ts", ".py", ".go", ".java",
			".ncm",
		}
	}
	return nil
}
