package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App         AppConfig         `mapstructure:"app"`
	Server      ServerConfig      `mapstructure:"server"`
	Database    DatabaseConfig    `mapstructure:"database"`
	Redis       RedisConfig       `mapstructure:"redis"`
	Logger      LoggerConfig      `mapstructure:"logger"`
	SMTP        SMTPConfig        `mapstructure:"smtp"`
	JWT         JWTConfig         `mapstructure:"jwt"`
	Kafka       KafkaConfig       `mapstructure:"kafka"`
	MinIO       MinIOConfig       `mapstructure:"minio"`
	AuthGRPC    AuthGRPCConfig    `mapstructure:"auth_grpc"`
	ProblemGRPC ProblemGRPCConfig `mapstructure:"problem_grpc"`
	JudgeGRPC   JudgeGRPCConfig   `mapstructure:"judge_grpc"`
	RunCode     RunCodeConfig     `mapstructure:"run_code"`
	SSE         SSEConfig         `mapstructure:"sse"`
}

type AppConfig struct {
	FrontendURL string `mapstructure:"frontend_url"`
}

type ServerConfig struct {
	Name     string `mapstructure:"name"`
	Port     int    `mapstructure:"port"`
	GRPCPort int    `mapstructure:"grpc_port"`
	Mode     string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"dbname"`
	SSLMode         string `mapstructure:"sslmode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type LoggerConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

type KafkaConfig struct {
	Brokers       string `mapstructure:"brokers"`
	JobTopic      string `mapstructure:"job_topic"`
	ResultTopic   string `mapstructure:"result_topic"`
	DLTTopic      string `mapstructure:"dlt_topic"`
	ConsumerGroup string `mapstructure:"consumer_group"`
}

type MinIOConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	UseSSL    bool   `mapstructure:"use_ssl"`
	Bucket    string `mapstructure:"bucket"`
	PublicURL string `mapstructure:"public_url"`
}

type AuthGRPCConfig struct {
	Address string        `mapstructure:"address"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type ProblemGRPCConfig struct {
	Address string        `mapstructure:"address"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type JudgeGRPCConfig struct {
	Address string        `mapstructure:"address"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type RunCodeConfig struct {
	MaxTestCases            int           `mapstructure:"max_testcases"`
	MaxSourceCodeBytes      int           `mapstructure:"max_source_code_bytes"`
	MaxStdinBytes           int           `mapstructure:"max_stdin_bytes"`
	MaxExpectedOutputBytes  int           `mapstructure:"max_expected_output_bytes"`
	MaxCapturedOutputBytes  int           `mapstructure:"max_captured_output_bytes"`
	MaxConcurrentRequests   int           `mapstructure:"max_concurrent_requests"`
	DefaultTimeLimit        time.Duration `mapstructure:"default_time_limit"`
	DefaultMemoryLimitKB    int64         `mapstructure:"default_memory_limit_kb"`
	DefaultOutputLimitBytes int64         `mapstructure:"default_output_limit_bytes"`
	CompileTimeLimit        time.Duration `mapstructure:"compile_time_limit"`
	CompileMemoryLimitKB    int64         `mapstructure:"compile_memory_limit_kb"`
	RequestTimeout          time.Duration `mapstructure:"request_timeout"`
}

type SSEConfig struct {
	TicketSecret      string        `mapstructure:"ticket_secret"`
	TicketTTL         time.Duration `mapstructure:"ticket_ttl"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	AllowedOrigin     string        `mapstructure:"allowed_origin"`
}

type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	FromName string `mapstructure:"from_name"`
	From     string `mapstructure:"from"`
}

type JWTConfig struct {
	AccessSecret  string        `mapstructure:"access_secret"`
	RefreshSecret string        `mapstructure:"refresh_secret"`
	AccessTTL     time.Duration `mapstructure:"access_ttl"`
	RefreshTTL    time.Duration `mapstructure:"refresh_ttl"`
}

func LoadConfig(path string) (*Config, error) {
	v := viper.New()
	v.AddConfigPath(path)
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
