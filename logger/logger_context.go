package logger

import (
	"context"

	"github.com/sirupsen/logrus"
)

// tenantIdCtxKeyType defines the type for the tenant's key that will ensure type safety when storing or fetching the
// variable to/from the context.
type tenantIdCtxKeyType string

// sourceIdCtxKeyType defines the type for the source's key that will ensure type safety when storing or fetching the
// variable to/from the context.
type sourceIdCtxKeyType string

// applicationIdCtxKeyType defines the type for the application's key that will ensure type safety when storing or
// fetching the variable to/from the context.
type applicationIdCtxKeyType string

// applicationTypeCtxKeyType defines the type for the application type's key that will ensure type safety when storing
// or fetching the variable to/from the context.
type applicationTypeCtxKeyType string

// tenantIdCtxKey defines the key to be used to store the tenant's identifier.
const tenantIdCtxKey tenantIdCtxKeyType = "tenant_id"

// sourceIdCtxKey defines the key to be used to store the source's identifier.
const sourceIdCtxKey sourceIdCtxKeyType = "source_id"

// applicationIdCtxKey defines the key to be used to store the application's identifier.
const applicationIdCtxKey applicationIdCtxKeyType = "application_id"

// applicationTypeCtxKey defines the key to be used to store the application type's identifier.
const applicationTypeCtxKey applicationTypeCtxKeyType = "application_type"

// LogWithContext returns a logger with all the fields defined in the context.
func LogWithContext(ctx context.Context) *logrus.Entry {
	logFields := logrus.Fields{}

	if tenantId, ok := ctx.Value(tenantIdCtxKey).(tenantIdCtxKeyType); ok {
		logFields["tenant_id"] = tenantId
	}

	if sourceId, ok := ctx.Value(sourceIdCtxKey).(sourceIdCtxKeyType); ok {
		logFields["source_id"] = sourceId
	}

	if applicationId, ok := ctx.Value(applicationIdCtxKey).(applicationIdCtxKeyType); ok {
		logFields["application_id"] = applicationId
	}

	if applicationType, ok := ctx.Value(applicationTypeCtxKey).(applicationTypeCtxKeyType); ok {
		logFields["application_type"] = applicationType
	}

	return Log.WithFields(logFields)
}

// WithTenantId creates a new context by copying the given context and appending the tenant's identifier to it.
func WithTenantId(ctx context.Context, tenantId string) context.Context {
	return context.WithValue(ctx, tenantIdCtxKey, tenantId)
}

// WithSourceId creates a new context by copying the given context and appending the source's identifier to it.
func WithSourceId(ctx context.Context, sourceId string) context.Context {
	return context.WithValue(ctx, sourceIdCtxKey, sourceId)
}

// WithApplicationId creates a new context by copying the given context and appending the application's identifier to
// it.
func WithApplicationId(ctx context.Context, applicationId string) context.Context {
	return context.WithValue(ctx, applicationIdCtxKey, applicationId)
}

// WithApplicationType creates a new context by copying the given context and appending the application's type to it.
func WithApplicationType(ctx context.Context, applicationType string) context.Context {
	return context.WithValue(ctx, applicationTypeCtxKey, applicationType)
}
