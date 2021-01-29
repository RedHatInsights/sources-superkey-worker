package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/segmentio/kafka-go"
)

func main() {
	var topic, key, value string
	var count int
	flag.StringVar(&topic, "topic", "testtopic", "the topic to produce to")
	flag.StringVar(&key, "key", "k", "the key of the message")
	flag.StringVar(&value, "value", "v", "the value of the message")
	flag.IntVar(&count, "count", 1, "how many times to repeat the message")
	flag.Parse()

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   topic,
	})

	msgs := make([]kafka.Message, 0, count)

	for i := 0; i < count; i++ {
		msgs = append(msgs, kafka.Message{
			Key:   []byte(key),
			Value: []byte(value),
			Headers: []kafka.Header{
				{Key: "hk", Value: []byte("hv")},
			},
		},
		)
	}

	err := w.WriteMessages(context.Background(), msgs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AAA failed %e\n", err)
		os.Exit(1)
	}
}
