package main

import (
	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/messaging"
	"github.com/segmentio/kafka-go"
)

// SuperKeyRequestQueue - the queue to listen on for superkey requests
var SuperKeyRequestQueue = "platform.sources.superkey-requests"

func main() {
	l.InitLogger()
	l.Log.Info("SuperKey Worker started.")

	// anonymous function, kinda like passing a block in ruby.
	messaging.ConsumeWithFunction(SuperKeyRequestQueue, func(msg kafka.Message) {
		l.Log.Infof("Started processing message %s", string(msg.Value))
		ProcessSuperkeyRequest(msg)
		l.Log.Infof("Finished processing message %s", string(msg.Value))
	})
}
