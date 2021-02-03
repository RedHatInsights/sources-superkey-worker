package provider

import (
	"fmt"

	"github.com/redhatinsights/sources-superkey-worker/amazon"
	"github.com/redhatinsights/sources-superkey-worker/sources"
)

// Forge -
func Forge(request *SuperKeyRequest) (*ForgedApplication, error) {
	client, err := getProvider(request)
	if err != nil {
		return nil, err
	}
	f, err := client.ForgeApplication(request)

	// returning both every time. we need the state of the forged product to know what to
	// tear down.
	return f, err
}

func getProvider(request *SuperKeyRequest) (SuperKeyProvider, error) {
	auth, err := sources.GetInternalAuthentication(request.TenantID, request.AuthenticationID)
	if err != nil {
		return nil, err
	}

	switch request.Provider {
	case "amazon":
		// client, err := amazon.NewClient(os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"), getStepNames(request.SuperKeySteps)...)
		client, err := amazon.NewClient(
			*auth.Username,
			*auth.Password,
			getStepNames(request.SuperKeySteps)...,
		)
		if err != nil {
			return nil, err
		}

		return &AmazonProvider{Client: client}, nil
	default:
		return nil, fmt.Errorf("Unsupported auth provider %v", request.Provider)
	}
}

func getStepNames(steps []SuperKeyStep) []string {
	names := make([]string, 0)
	for _, step := range steps {
		names = append(names, step.Name)
	}

	return names
}

// TearDown - tears down application that was forged
// returns: array of errors if any were returned.
func TearDown(f *ForgedApplication) []error {
	if f.Client != nil {
		return f.Client.TearDown(f)
	}

	return []error{}
}
