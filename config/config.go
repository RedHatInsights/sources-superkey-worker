package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// SuperKeyWorkerConfig is the struct for storing runtime configuration
type SuperKeyWorkerConfig struct {
	Hostname           string
	KafkaBrokers       []string
	KafkaGroupID       string
	MetricsPort        int
	LogLevel           string
	LogGroup           string
	AwsRegion          string
	AwsAccessKeyID     string
	AwsSecretAccessKey string
	SourcesHost        string
	SourcesScheme      string
	SourcesPort        int
}

// Get - returns the config parsed from runtime vars
func Get() *SuperKeyWorkerConfig {
	options := viper.New()

	options.SetDefault("KafkaBrokers", []string{fmt.Sprintf("%v:%v", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT"))})
	options.SetDefault("KafkaGroupID", "sources-superkey-worker")
	options.SetDefault("MetricsPort", 9394)
	options.SetDefault("LogLevel", "INFO")
	options.SetDefault("LogGroup", os.Getenv("CLOUD_WATCH_LOG_GROUP"))
	options.SetDefault("AwsRegion", "us-east-1")
	options.SetDefault("AwsAccessKeyId", os.Getenv("CW_AWS_ACCESS_KEY_ID"))
	options.SetDefault("AwsSecretAccessKey", os.Getenv("CW_AWS_SECRET_ACCESS_KEY"))
	options.SetDefault("SourcesHost", os.Getenv("SOURCES_HOST"))
	options.SetDefault("SourcesScheme", os.Getenv("SOURCES_SCHEME"))
	options.SetDefault("SourcesPort", os.Getenv("SOURCES_PORT"))

	hostname, _ := os.Hostname()
	options.SetDefault("Hostname", hostname)

	options.AutomaticEnv()

	return &SuperKeyWorkerConfig{
		Hostname:           options.GetString("Hostname"),
		KafkaBrokers:       options.GetStringSlice("KafkaBrokers"),
		KafkaGroupID:       options.GetString("KafkaGroupID"),
		MetricsPort:        options.GetInt("MetricsPort"),
		LogLevel:           options.GetString("LogLevel"),
		LogGroup:           options.GetString("LogGroup"),
		AwsRegion:          options.GetString("AwsRegion"),
		AwsAccessKeyID:     options.GetString("AwsAccessKeyID"),
		AwsSecretAccessKey: options.GetString("AwsSecretAccessKey"),
		SourcesHost:        options.GetString("SourcesHost"),
		SourcesScheme:      options.GetString("SourcesScheme"),
		SourcesPort:        options.GetInt("SourcesPort"),
	}
}
