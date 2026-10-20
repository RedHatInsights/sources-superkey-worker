package logger

// NewCustomLoggerFormatter adds hostname and app name
type CustomLoggerFormatter struct {
	Hostname string
	AppName  string
}

// Marshaler is an interface any type can implement to change its output in our production logs.
type Marshaler interface {
	MarshalLog() map[string]interface{}
}
