package provider

import (
	"context"
	"fmt"

	"github.com/redhatinsights/sources-superkey-worker/amazon"
	"github.com/redhatinsights/sources-superkey-worker/config"
	"github.com/redhatinsights/sources-superkey-worker/sources"
	"github.com/redhatinsights/sources-superkey-worker/superkey"
)

// Forge - creates the provider client based on provider and forges resources
func Forge(ctx context.Context, request *superkey.CreateRequest) (*superkey.ForgedApplication, error) {
	client, err := getProvider(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("unable to get provider: %w", err)
	}

	f, err := client.ForgeApplication(ctx, request)

	// returning both every time. we need the state of the forged product to know what to
	// tear down.
	return f, err
}

// TearDown - tears down application that was forged
// returns: array of errors if any were returned.
func TearDown(ctx context.Context, f *superkey.ForgedApplication) []error {
	if f == nil {
		return []error{}
	}

	// the client is nil if it came from a destroy request
	if f.Client == nil {
		client, err := getProvider(ctx, f.Request)
		if err != nil {
			return []error{err}
		}

		f.Client = client
	}

	return f.Client.TearDown(ctx, f)
}

// getProvider returns a provider based on create request's provider + credentials
func getProvider(ctx context.Context, request *superkey.CreateRequest) (superkey.Provider, error) {
	sourcesRestClient := sources.NewSourcesClient(config.Get())

	authData := sources.AuthenticationData{
		IdentityHeader: request.IdentityHeader,
		OrgId:          request.OrgIdHeader,
	}

	auth, err := sourcesRestClient.GetInternalAuthentication(ctx, &authData, request.SuperKey)
	if err != nil {
		return nil, fmt.Errorf(`error while fetching internal authentication "%s" from Sources: %w`, request.SuperKey, err)
	}

	if auth.Username == "" || auth.Password == "" {
		return nil, fmt.Errorf(`missing username or password from authentication ID "%s" and superkey credential "%s"`, auth.ID, request.SuperKey)
	}

	switch request.Provider {
	case "amazon":
		client, err := amazon.NewClient(ctx, auth.Username, auth.Password, getStepNames(request.SuperKeySteps)...)
		if err != nil {
			return nil, fmt.Errorf(`unable to create Amazon client with authentication ID "%s": %w`, auth.ID, err)
		}

		return &AmazonProvider{Client: client}, nil
	default:
		return nil, fmt.Errorf(`unsupported auth provider "%s"`, request.Provider)
	}
}

func getStepNames(steps []superkey.Step) []string {
	names := make([]string, 0)
	for _, step := range steps {
		names = append(names, step.Name)
	}

	return names
}
