package main

import (
	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/messaging"
	"github.com/segmentio/kafka-go"
)

func main() {
	l.InitLogger()
	l.Log.Infoln("SuperKey Worker started.")

	// anonymous function, kinda like passing a block in ruby.
	messaging.ConsumeWithFunction("testtopic", func(msg kafka.Message) {
		l.Log.Tracef("Started processing message %s", string(msg.Value))
		ProcessSuperkeyRequest(msg)
		l.Log.Tracef("Finished processing message %s", string(msg.Value))
	})
}
