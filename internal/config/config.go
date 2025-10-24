package config

import (
	"fmt"
	"os"
	"strconv"
	// "gopkg.in/yaml.v3"
)

type Config struct {
	DB       *DBconfig
	RabbitMq *RabbitMqconfig
	WS       *WebSocketconfig
	Srv      *Serviceconfig
	Log      *Loggerconfig
	App      *App
}

type DBconfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type RabbitMqconfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	VHost    string `yaml:"vhost"`
}

type WebSocketconfig struct {
	Port int `yaml:"port"`
}

type Serviceconfig struct {
	RideServicePort           string `yaml:"ride_service"`
	DriverLocationServicePort string `yaml:"driver_location_service"`
	AdminServicePort          string `yaml:"admin_service"`
	AuthServicePort           string `yaml:"auth_service"`
}

type Loggerconfig struct {
	Level string `yaml:"level"`
}

type App struct {
	PublicJwtSecret  string `yaml:"public_jwt"`
	PrivateJwtSecret string `yaml:"private_jwt"`
}

func New() (*Config, error) {
	getEnv := func(key, def string) string {
		val := os.Getenv(key)
		if val == "" {
			fmt.Printf("using default key %v\n", def)
			return def
		}
		return val
	}

	getEnvInt := func(key string, def int) int {
		valStr := os.Getenv(key)
		if valStr == "" {
			fmt.Printf("using default key %v\n", def)
			return def
		}
		val, err := strconv.Atoi(valStr)
		if err != nil {
			fmt.Printf("using default key %v\n", def)
			return def
		}
		return val
	}

	cnf := &Config{
		DB: &DBconfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "ridehail_user"),
			Password: getEnv("DB_PASSWORD", "ridehail_pass"),
			Database: getEnv("DB_NAME", "ridehail_db"),
		},
		RabbitMq: &RabbitMqconfig{
			Host:     getEnv("RABBITMQ_HOST", "localhost"),
			Port:     getEnvInt("RABBITMQ_PORT", 5672),
			User:     getEnv("RABBITMQ_USER", "admin"),
			Password: getEnv("RABBITMQ_PASSWORD", "admin"),
			VHost:    getEnv("RABBITMQ_VHOST", "fake-taxi"),
		},
		WS: &WebSocketconfig{
			Port: getEnvInt("WS_PORT", 8080),
		},
		Srv: &Serviceconfig{
			RideServicePort:           getEnv("RIDE_SERVICE_PORT", "3002"),
			DriverLocationServicePort: getEnv("DRIVER_LOCATION_SERVICE_PORT", "3001"),
			AdminServicePort:          getEnv("ADMIN_SERVICE_PORT", "3004"),
			AuthServicePort:           getEnv("AUTH_SERVICE_PORT", "3010"),
		},
		Log: &Loggerconfig{
			Level: getEnv("LOG_LEVEL", "INFO"),
		},
		App: &App{
			PublicJwtSecret:  getEnv("PUBLIC_JWT", "default-public-secret"),
			PrivateJwtSecret: getEnv("PRIVATE_JWT", "default-private-secret"),
		},
	}

	return cnf, nil
}

// func NewFromYAML(path string) (*Config, error) {
// 	data, err := os.ReadFile(path)
// 	if err != nil {
// 		return nil, err
// 	}

// 	cnf := &Config{}
// 	if err := yaml.Unmarshal(data, cnf); err != nil {
// 		return nil, err
// 	}

// 	return cnf, nil
// }
