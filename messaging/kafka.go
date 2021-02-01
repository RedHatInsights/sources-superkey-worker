package messaging

import (
	"context"

	"github.com/redhatinsights/sources-superkey-worker/config"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/segmentio/kafka-go"
)

// Consumer creates a consumer for a topic
// returns: a kafka reader configured to listen on topic with groupID set
func Consumer(topic string) *kafka.Reader {
	conf := config.Get()

	return kafka.NewReader(kafka.ReaderConfig{
		Brokers: conf.KafkaBrokers,
		GroupID: conf.KafkaGroupID,
		Topic:   topic,
	})
}

// ConsumeWithFunction consumes on topic calling function on each message
// consumed
func ConsumeWithFunction(topic string, whatdo func(msg kafka.Message)) {
	r := Consumer(topic)

	for {
		// ReadMessage blocks until a message is read.
		msg, err := r.ReadMessage(context.Background())
		if err != nil {
			l.Log.Errorf("Error while processing message: %v", err)
			continue
		}

		go whatdo(msg)
	}
}
