package config

import (
	"log"
	"notification-service/internal/notification"

	"github.com/spf13/viper"
)

type Config struct {
	HTTPAddr     string                      `mapstructure:"http_addr"`
	Email        notification.EmailConfig    `mapstructure:"-"`
	Slack        notification.SlackConfig    `mapstructure:"-"`
	InApp        notification.InAppConfig    `mapstructure:"-"`
	Notification NotificationConfig          `mapstructure:"notification"`
}

type NotificationConfig struct {
	Email notification.EmailConfig `mapstructure:"email"`
	Slack notification.SlackConfig `mapstructure:"slack"`
	InApp notification.InAppConfig `mapstructure:"inapp"`
}

func LoadConfig() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config") // config.yaml lives alongside this file

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: error reading config file: %s", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}

	// For backward compatibility or ease of use in app.go
	config.Email = config.Notification.Email
	config.Slack = config.Notification.Slack
	config.InApp = config.Notification.InApp

	return &config
}
