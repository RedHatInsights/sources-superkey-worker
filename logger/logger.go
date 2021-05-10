package logger

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	lc "github.com/redhatinsights/platform-go-middlewares/logging/cloudwatch"
	appconf "github.com/redhatinsights/sources-superkey-worker/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Log is an instance of the global logrus.Logger
var Log *logrus.Logger
var logLevel logrus.Level

// NewCustomLoggerFormatter creates a new log formatter
func NewCustomLoggerFormatter() *CustomLoggerFormatter {
	f := &CustomLoggerFormatter{AppName: "sources-superkey-worker"}

	var err error
	if f.Hostname == "" {
		if f.Hostname, err = os.Hostname(); err != nil {
			f.Hostname = "unknown"
		}
	}

	return f
}

//Format is the log formatter for the entry
func (f *CustomLoggerFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	b := &bytes.Buffer{}

	now := time.Now()

	hostname, err := os.Hostname()
	if err == nil {
		f.Hostname = hostname
	}

	data := map[string]interface{}{
		"@timestamp": now.Format("2006-01-02T15:04:05.999Z"),
		"@version":   1,
		"message":    entry.Message,
		"levelname":  entry.Level.String(), //Backward compatibility - TODO: remove?
		"level":      entry.Level.String(),
		"hostname":   f.Hostname,
		"app":        f.AppName,
		"caller":     entry.Caller.Func.Name(),
		"labels":     map[string]interface{}{"app" : f.AppName},
		"tags":       []string{f.AppName},
	}

	for k, v := range entry.Data {
		switch v := v.(type) {
		case error:
			data[k] = v.Error()
		case Marshaler:
			data[k] = v.MarshalLog()
		default:
			data[k] = v
		}
	}

	j, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	b.Write(j)

    b.Write([]byte("\n"))
	return b.Bytes(), nil
}

func ForwardLogsToStderr(logHandler string) bool {
    return logHandler == "haberdasher"
}

// InitLogger initializes the Sources SuperKey logger
func InitLogger(cfg *appconf.SuperKeyWorkerConfig) *logrus.Logger {
	logconfig := viper.New()

	key := cfg.AwsAccessKeyID
	secret := cfg.AwsSecretAccessKey
	region := cfg.AwsRegion
	group := cfg.LogGroup
	stream := cfg.Hostname
	logconfig.SetEnvPrefix("SSKWORKER")
	logconfig.AutomaticEnv()

	switch cfg.LogLevel {
	case "DEBUG":
		logLevel = logrus.DebugLevel
	case "ERROR":
		logLevel = logrus.ErrorLevel
	default:
		logLevel = logrus.InfoLevel
	}
	if flag.Lookup("test.v") != nil {
		logLevel = logrus.FatalLevel
	}

	formatter := NewCustomLoggerFormatter()

    logOutput := os.Stdout
    forwardLogsToStderr := ForwardLogsToStderr(cfg.LogHandler)

    if forwardLogsToStderr {
        logOutput = os.Stderr
    }

	Log = &logrus.Logger{
		Out:          logOutput,
		Level:        logLevel,
		Formatter:    formatter,
		Hooks:        make(logrus.LevelHooks),
		ReportCaller: true,
	}

    if forwardLogsToStderr {
        return Log
    }

	// TODO: maybe redo this to work with the go-aws-v2 library.
	// That would involve updating the platform middleware though, which might
	// not be fun.
	if key != "" && secret != "" {
		cred := credentials.NewStaticCredentials(key, secret, "")
		awsconf := aws.NewConfig().WithRegion(region).WithCredentials(cred)
		hook, err := lc.NewBatchingHook(group, stream, awsconf, 10*time.Second)
		if err != nil {
			Log.Info(err)
		}
		Log.Hooks.Add(hook)
	}

	return Log
}
