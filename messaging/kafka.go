package messaging

import (
	"context"

	"github.com/redhatinsights/sources-superkey-worker/config"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/segmentio/kafka-go"
)

// Consumer - I'm putting this comment just so I don't get fined by the linter
func Consumer(topic string) *kafka.Reader {
	conf := config.Get()

	return kafka.NewReader(kafka.ReaderConfig{
		Brokers: conf.KafkaBrokers,
		GroupID: conf.KafkaGroupID,
		Topic:   topic,
	})
}

// ConsumeWithFunction testing
func ConsumeWithFunction(topic string, whatdo func(msg kafka.Message)) {
	r := Consumer(topic)

	for {
		msg, err := r.ReadMessage(context.Background())
		if err != nil {
			l.Log.Errorf("Error while processing message: %e", err)
			continue
		}

		go whatdo(msg)
	}
}
