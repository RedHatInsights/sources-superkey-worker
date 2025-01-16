package logger

import (
	"context"
	"net/http"
	"net/url"

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

// authenticationIdCtxKeyType defines the type for the authentication's identifier key that will ensure type safety
// when storing or fetching the variable to/from the context.
type authenticationIdCtxKeyType string

// resourceTypeCtxKeyType defines the type for the resource's type key that will ensure type safety when storing or
// fetching the variable to/from the context.
type resourceTypeCtxKeyType string

// resourceIdCtxKeyType defines the type for the resource's identifier key that will ensure type safety when storing or
// fetching the variable to/from the context.
type resourceIdCtxKeyType string

// httpMethodCtxKeyType defines the type for the HTTP method key that will ensure type safety when storing or
// fetching the variable to/from the context.
type httpMethodCtxKeyType string

// urlCtxKeyType defines the type for the URL key that will ensure type safety when storing or fetching the variable
// to/from the context.
type urlCtxKeyType string

// urlCtxKeyType defines the type for the HTTP headers key that will ensure type safety when storing or fetching the
// variable to/from the context.
type httpHeadersCtxKeyType string

// tenantIdCtxKey defines the key to be used to store the tenant's identifier.
const tenantIdCtxKey tenantIdCtxKeyType = "tenant_id"

// sourceIdCtxKey defines the key to be used to store the source's identifier.
const sourceIdCtxKey sourceIdCtxKeyType = "source_id"

// applicationIdCtxKey defines the key to be used to store the application's identifier.
const applicationIdCtxKey applicationIdCtxKeyType = "application_id"

// applicationTypeCtxKey defines the key to be used to store the application type's identifier.
const applicationTypeCtxKey applicationTypeCtxKeyType = "application_type"

// authenticationIdCtxKey defines the key to be used to store the authentication's identifier.
const authenticationIdCtxKey authenticationIdCtxKeyType = "authentication_id"

// resourceTypeCtxKey defines the key to be used to store the resource's type.
const resourceTypeCtxKey resourceTypeCtxKeyType = "resource_type"

// resourceIdCtxKey defines the key to be used to store the resource's identifier.
const resourceIdCtxKey resourceIdCtxKeyType = "resource_id"

// httpMethodCtxKey defines the key to be used to store the HTTP method's identifier.
const httpMethodCtxKey httpMethodCtxKeyType = "http_method"

// urlCtxKey defines the key to be used to store the URL's identifier.
const urlCtxKey urlCtxKeyType = "url"

// httpHeadersCtxKey defines the key to be used to store the HTTP headers' identifier.
const httpHeadersCtxKey httpHeadersCtxKeyType = "http_headers"

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

	if authenticationId, ok := ctx.Value(authenticationIdCtxKey).(authenticationIdCtxKeyType); ok {
		logFields["authentication_id"] = authenticationId
	}

	if resourceType, ok := ctx.Value(resourceTypeCtxKey).(resourceTypeCtxKeyType); ok {
		logFields["resource_type"] = resourceType
	}

	if resourceId, ok := ctx.Value(resourceIdCtxKey).(resourceIdCtxKeyType); ok {
		logFields["resource_id"] = resourceId
	}

	if httpMethod, ok := ctx.Value(httpMethodCtxKey).(authenticationIdCtxKeyType); ok {
		logFields["http_method"] = httpMethod
	}

	if urlVar, ok := ctx.Value(urlCtxKey).(urlCtxKeyType); ok {
		logFields["url"] = urlVar
	}

	if httpHeaders, ok := ctx.Value(httpHeadersCtxKey).(httpHeadersCtxKeyType); ok {
		logFields["http_headers"] = httpHeaders
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

// WithAuthenticationId creates a new context by copying the given context and appending the authentication's
// identifier to it.
func WithAuthenticationId(ctx context.Context, authenticationId string) context.Context {
	return context.WithValue(ctx, authenticationIdCtxKey, authenticationId)
}

// WithResourceType creates a new context by copying the given context and appending the resource's type to it.
func WithResourceType(ctx context.Context, resourceType string) context.Context {
	return context.WithValue(ctx, resourceTypeCtxKey, resourceType)
}

// WithResourceId creates a new context by copying the given context and appending the resource's identifier to it.
func WithResourceId(ctx context.Context, resourceId string) context.Context {
	return context.WithValue(ctx, resourceIdCtxKey, resourceId)
}

// WithHttpMethod creates a new context by copying the given context and appending the HTTP method to it.
func WithHttpMethod(ctx context.Context, httpMethod string) context.Context {
	return context.WithValue(ctx, httpMethodCtxKey, httpMethod)
}

// WithURL creates a new context by copying the given context and appending the URL to it.
func WithURL(ctx context.Context, url *url.URL) context.Context {
	return context.WithValue(ctx, urlCtxKey, url.String())
}

// WithHTTPHeaders creates a new context by copying the given context and appending the HTTP headers to it.
func WithHTTPHeaders(ctx context.Context, headers http.Header) context.Context {
	return context.WithValue(ctx, httpHeadersCtxKey, headers)
}
