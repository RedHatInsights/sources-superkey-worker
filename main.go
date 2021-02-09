package main

import (
	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/messaging"
	"github.com/segmentio/kafka-go"
)

func main() {
	l.InitLogger()
	l.Log.Info("SuperKey Worker started.")

	// anonymous function, kinda like passing a block in ruby.
	messaging.ConsumeWithFunction("platform.sources.superkey-requests", func(msg kafka.Message) {
		l.Log.Infof("Started processing message %s", string(msg.Value))
		ProcessSuperkeyRequest(msg)
		l.Log.Infof("Finished processing message %s", string(msg.Value))
	})
}
