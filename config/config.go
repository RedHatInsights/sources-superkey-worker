package config

import (
	"log"
	"os"
	"strconv"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/spf13/viper"
)

// SuperKeyWorkerConfig is the struct for storing runtime configuration
type SuperKeyWorkerConfig struct {
	Hostname                   string
	KafkaBrokerConfig          []clowder.BrokerConfig
	KafkaTopics                map[string]string
	KafkaGroupID               string
	MetricsPort                int
	LogLevel                   string
	LogGroup                   string
	LogHandler                 string
	AwsRegion                  string
	AwsAccessKeyID             string
	AwsSecretAccessKey         string
	SourcesHost                string
	SourcesScheme              string
	SourcesPort                int
	SourcesPSK                 string
	SourcesRequestsMaxAttempts int
}

// Get - returns the config parsed from runtime vars
func Get() *SuperKeyWorkerConfig {
	options := viper.New()
	kafkaTopics := make(map[string]string)

	if clowder.IsClowderEnabled() {
		cfg := clowder.LoadedConfig

		for requestedName, topicConfig := range clowder.KafkaTopics {
			kafkaTopics[requestedName] = topicConfig.Name
		}
		options.SetDefault("AwsRegion", cfg.Logging.Cloudwatch.Region)
		options.SetDefault("AwsAccessKeyId", cfg.Logging.Cloudwatch.AccessKeyId)
		options.SetDefault("AwsSecretAccessKey", cfg.Logging.Cloudwatch.SecretAccessKey)
		options.SetDefault("KafkaBrokerConfig", cfg.Kafka.Brokers)
		options.SetDefault("LogGroup", cfg.Logging.Cloudwatch.LogGroup)
		options.SetDefault("MetricsPort", cfg.MetricsPort)

	} else {
		options.SetDefault("AwsRegion", "us-east-1")
		options.SetDefault("AwsAccessKeyId", os.Getenv("CW_AWS_ACCESS_KEY_ID"))
		options.SetDefault("AwsSecretAccessKey", os.Getenv("CW_AWS_SECRET_ACCESS_KEY"))

		kafkaPort := os.Getenv("QUEUE_PORT")
		if kafkaPort != "" {
			port, err := strconv.Atoi(kafkaPort)
			if err != nil {
				log.Fatalf(`the provided "QUEUE_PORT", "%s", is not a valid integer: %s`, kafkaPort, err)
			}

			brokerConfig := []clowder.BrokerConfig{{
				Hostname: os.Getenv("QUEUE_HOST"),
				Port:     &port,
			}}

			options.SetDefault("KafkaBrokerConfig", brokerConfig)
		}

		options.SetDefault("LogGroup", os.Getenv("CLOUD_WATCH_LOG_GROUP"))
		options.SetDefault("MetricsPort", 9394)

	}

	options.SetDefault("KafkaGroupID", "sources-superkey-worker")
	options.SetDefault("KafkaTopics", kafkaTopics)
	options.SetDefault("LogLevel", os.Getenv("LOG_LEVEL"))
	options.SetDefault("LogHandler", os.Getenv("LOG_HANDLER"))

	options.SetDefault("SourcesHost", os.Getenv("SOURCES_HOST"))
	options.SetDefault("SourcesScheme", os.Getenv("SOURCES_SCHEME"))
	options.SetDefault("SourcesPort", os.Getenv("SOURCES_PORT"))
	options.SetDefault("SourcesPSK", os.Getenv("SOURCES_PSK"))

	// Get the number of maximum request attempts we want to make to the Sources' API.
	sourcesRequestsMaxAttempts, err := strconv.Atoi(os.Getenv("SOURCES_MAX_ATTEMPTS"))
	if err != nil {
		log.Printf(`Warning: the provided max attempts value \"%s\" is not an integer. Setting default value of 1.`, os.Getenv("SOURCES_MAX_ATTEMPTS"))
		sourcesRequestsMaxAttempts = 1
	}

	if sourcesRequestsMaxAttempts < 1 {
		log.Printf(`Warning: the provided max attempts value \"%s\" is lower than 1, and we need to at least make one attempt when calling Sources. Setting default value of 1.`, os.Getenv("SOURCES_MAX_ATTEMPTS"))
		sourcesRequestsMaxAttempts = 1
	}

	options.SetDefault("SourcesRequestsMaxAttempts", sourcesRequestsMaxAttempts)

	hostname, _ := os.Hostname()
	options.SetDefault("Hostname", hostname)

	// Grab the Kafka broker configuration Settings.
	var brokerConfig []clowder.BrokerConfig
	bcRaw, ok := options.Get("KafkaBrokerConfig").([]clowder.BrokerConfig)
	if ok {
		brokerConfig = bcRaw
	}

	options.AutomaticEnv()

	return &SuperKeyWorkerConfig{
		Hostname:                   options.GetString("Hostname"),
		KafkaBrokerConfig:          brokerConfig,
		KafkaTopics:                options.GetStringMapString("KafkaTopics"),
		KafkaGroupID:               options.GetString("KafkaGroupID"),
		MetricsPort:                options.GetInt("MetricsPort"),
		LogLevel:                   options.GetString("LogLevel"),
		LogHandler:                 options.GetString("LogHandler"),
		LogGroup:                   options.GetString("LogGroup"),
		AwsRegion:                  options.GetString("AwsRegion"),
		AwsAccessKeyID:             options.GetString("AwsAccessKeyID"),
		AwsSecretAccessKey:         options.GetString("AwsSecretAccessKey"),
		SourcesHost:                options.GetString("SourcesHost"),
		SourcesScheme:              options.GetString("SourcesScheme"),
		SourcesPort:                options.GetInt("SourcesPort"),
		SourcesPSK:                 options.GetString("SourcesPSK"),
		SourcesRequestsMaxAttempts: options.GetInt("SourcesRequestsMaxAttempts"),
	}
}

func (s *SuperKeyWorkerConfig) KafkaTopic(topic string) string {
	found, ok := s.KafkaTopics[topic]
	if ok {
		return found
	}

	return topic
}
