package main

import (
	"fmt"
	"strings"

	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"github.com/redhatinsights/sources-superkey-worker/messaging"
	"github.com/segmentio/kafka-go"
)

func main() {
	l.InitLogger()

	// anonymous function, kinda like passing a block in ruby.
	messaging.ConsumeWithFunction("testtopic", func(msg kafka.Message) {
		var sb strings.Builder
		for i := 0; i < len(msg.Headers); i++ {
			sb.Write([]byte(fmt.Sprintf("(%s:%s) ", msg.Headers[i].Key, msg.Headers[i].Value)))
		}

		l.Log.Infof("%s, %s, %s", msg.Key, msg.Value, sb.String())
	})
}
