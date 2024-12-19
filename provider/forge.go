package provider

import (
	"fmt"

	"github.com/redhatinsights/sources-superkey-worker/amazon"
	"github.com/redhatinsights/sources-superkey-worker/sources"
	"github.com/redhatinsights/sources-superkey-worker/superkey"
)

// Forge - creates the provider client based on provider and forges resources
func Forge(request *superkey.CreateRequest) (*superkey.ForgedApplication, error) {
	client, err := getProvider(request)
	if err != nil {
		return nil, fmt.Errorf("unable to get provider: %w", err)
	}

	f, err := client.ForgeApplication(request)

	// returning both every time. we need the state of the forged product to know what to
	// tear down.
	return f, err
}

// TearDown - tears down application that was forged
// returns: array of errors if any were returned.
func TearDown(f *superkey.ForgedApplication) []error {
	if f == nil {
		return []error{}
	}

	// the client is nil if it came from a destroy request
	if f.Client == nil {
		client, err := getProvider(f.Request)
		if err != nil {
			return []error{err}
		}

		f.Client = client
	}

	return f.Client.TearDown(f)
}

// getProvider returns a provider based on create request's provider + credentials
func getProvider(request *superkey.CreateRequest) (superkey.Provider, error) {
	client := sources.SourcesClient{AccountNumber: request.TenantID, IdentityHeader: request.IdentityHeader, OrgId: request.OrgIdHeader}
	auth, err := client.GetInternalAuthentication(request.TenantID, request.SourceID, request.SuperKey)
	if err != nil {
		return nil, fmt.Errorf(`error while fetching internal authentication "%s" from Sources: %w`, request.SuperKey, err)
	}

	if auth.Username == "" || auth.Password == "" {
		return nil, fmt.Errorf(`missing username or password from authentication ID "%s" and superkey credential "%s"`, auth.ID, request.SuperKey)
	}

	switch request.Provider {
	case "amazon":
		client, err := amazon.NewClient(auth.Username, auth.Password, getStepNames(request.SuperKeySteps)...)
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
